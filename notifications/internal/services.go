package internal

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/google/uuid"
	"thugcorp.io/grocery/notifications/internal/channels"
	"thugcorp.io/grocery/notifications/internal/domain"
	"thugcorp.io/grocery/notifications/internal/templates"
)

type NotificationService interface {
	Send(ctx context.Context, input domain.SendInput) (notificationID string, results []domain.ChannelResult, err error)
	ListNotifications(ctx context.Context, recipientID string, pageSize int, cursor string, unreadOnly bool) ([]*domain.Notification, string, error)
	MarkRead(ctx context.Context, recipientID string, ids []string) (int32, error)
	GetUnreadCount(ctx context.Context, recipientID string) (int32, error)
	GetPreferences(ctx context.Context, userID string) ([]*domain.NotificationPreference, error)
	UpdatePreferences(ctx context.Context, userID string, prefs []*domain.NotificationPreference) ([]*domain.NotificationPreference, error)
}

type notificationService struct {
	repo  NotificationRepository
	email channels.Dispatcher
	sms   channels.Dispatcher
	push  channels.Dispatcher
}

func NewNotificationService(repo NotificationRepository) NotificationService {
	return &notificationService{
		repo:  repo,
		email: channels.NewEmailDispatcher(),
		sms:   channels.NewSMSDispatcher(),
		push:  channels.NewPushDispatcher(),
	}
}

// ---- Send ----

func (s *notificationService) Send(ctx context.Context, input domain.SendInput) (string, []domain.ChannelResult, error) {
	// Idempotency: return early if this key was already processed.
	if input.IdempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, input.IdempotencyKey)
		if err != nil {
			return "", nil, err
		}
		if existing != nil {
			return existing.ID, []domain.ChannelResult{{Channel: "CHANNEL_IN_APP", Status: "DELIVERY_STATUS_SENT"}}, nil
		}
	}

	// Resolve template.
	tmpl := templates.Get(input.TemplateID)
	if tmpl == nil {
		return "", nil, errors.New("unknown template: " + input.TemplateID)
	}
	rendered := tmpl.Render(input.Variables)

	// Determine which channels to use.
	active := resolveChannels(input.Channels, rendered.DefaultChannels)

	// Filter against user preferences for non-transactional messages.
	if input.Category != "CATEGORY_TRANSACTIONAL" && input.RecipientUserID != "" {
		prefs, err := s.repo.GetPreferences(ctx, input.RecipientUserID)
		if err != nil {
			log.Printf("failed to load preferences for user %s: %v", input.RecipientUserID, err)
		} else {
			active = applyPreferences(active, input.Category, prefs)
		}
	}

	notificationID := uuid.New().String()
	var results []domain.ChannelResult

	for _, ch := range active {
		result := s.dispatch(ctx, ch, input, rendered, notificationID)
		results = append(results, result)
	}

	return notificationID, results, nil
}

func (s *notificationService) dispatch(
	ctx context.Context,
	ch string,
	input domain.SendInput,
	rendered *templates.RenderedTemplate,
	notificationID string,
) domain.ChannelResult {
	result := domain.ChannelResult{Channel: ch}

	var err error
	switch ch {
	case "CHANNEL_EMAIL":
		if input.Email == "" {
			result.Status = "DELIVERY_STATUS_SUPPRESSED"
			result.Detail = "no email address"
			return result
		}
		err = s.email.Send(ctx, input.Email, rendered.EmailSubject, rendered.EmailBody)

	case "CHANNEL_SMS":
		if input.Phone == "" {
			result.Status = "DELIVERY_STATUS_SUPPRESSED"
			result.Detail = "no phone number"
			return result
		}
		err = s.sms.Send(ctx, input.Phone, "", rendered.SMSBody)

	case "CHANNEL_PUSH":
		if input.PushToken == "" {
			result.Status = "DELIVERY_STATUS_SUPPRESSED"
			result.Detail = "no push token"
			return result
		}
		err = s.push.Send(ctx, input.PushToken, rendered.PushTitle, rendered.PushBody)

	case "CHANNEL_IN_APP":
		recipientID := input.RecipientUserID
		if recipientID == "" {
			recipientID = input.RecipientBusinessID
		}

		dataJSON, _ := json.Marshal(input.Variables)
		n := &domain.Notification{
			ID:             notificationID,
			RecipientID:    recipientID,
			Title:          rendered.InAppTitle,
			Body:           rendered.InAppBody,
			Category:       input.Category,
			Data:           string(dataJSON),
			IdempotencyKey: input.IdempotencyKey,
		}
		err = s.repo.Save(ctx, n)

	default:
		result.Status = "DELIVERY_STATUS_FAILED"
		result.Detail = "unknown channel: " + ch
		return result
	}

	if err != nil {
		log.Printf("[notifications] channel %s failed: %v", ch, err)
		result.Status = "DELIVERY_STATUS_FAILED"
		result.Detail = err.Error()
	} else {
		result.Status = "DELIVERY_STATUS_SENT"
	}
	return result
}

// ---- Inbox ----

func (s *notificationService) ListNotifications(ctx context.Context, recipientID string, pageSize int, cursor string, unreadOnly bool) ([]*domain.Notification, string, error) {
	return s.repo.List(ctx, recipientID, pageSize, cursor, unreadOnly)
}

func (s *notificationService) MarkRead(ctx context.Context, recipientID string, ids []string) (int32, error) {
	return s.repo.MarkRead(ctx, recipientID, ids)
}

func (s *notificationService) GetUnreadCount(ctx context.Context, recipientID string) (int32, error) {
	return s.repo.GetUnreadCount(ctx, recipientID)
}

// ---- Preferences ----

func (s *notificationService) GetPreferences(ctx context.Context, userID string) ([]*domain.NotificationPreference, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *notificationService) UpdatePreferences(ctx context.Context, userID string, incoming []*domain.NotificationPreference) ([]*domain.NotificationPreference, error) {
	for _, p := range incoming {
		p.UserID = userID
		if err := s.repo.UpsertPreference(ctx, p); err != nil {
			return nil, err
		}
	}
	return s.repo.GetPreferences(ctx, userID)
}

// ---- Helpers ----

// resolveChannels returns the requested channels, falling back to template defaults when empty.
func resolveChannels(requested, defaults []string) []string {
	if len(requested) > 0 {
		return requested
	}
	return defaults
}

// applyPreferences removes channels the user has opted out of for the given category.
func applyPreferences(channels []string, category string, prefs []*domain.NotificationPreference) []string {
	var allowed []string
	for _, p := range prefs {
		if p.Category == category {
			json.Unmarshal([]byte(p.Channels), &allowed)
			break
		}
	}
	if len(allowed) == 0 {
		return channels // no preference set — send on all
	}

	allowedSet := make(map[string]bool, len(allowed))
	for _, ch := range allowed {
		allowedSet[ch] = true
	}

	var filtered []string
	for _, ch := range channels {
		if allowedSet[ch] {
			filtered = append(filtered, ch)
		}
	}
	return filtered
}
