package internal

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/grocery/notifications/internal/domain"
)

type NotificationRepository interface {
	Save(ctx context.Context, n *domain.Notification) error
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error)
	List(ctx context.Context, recipientID string, pageSize int, cursor string, unreadOnly bool) ([]*domain.Notification, string, error)
	MarkRead(ctx context.Context, recipientID string, ids []string) (int32, error)
	GetUnreadCount(ctx context.Context, recipientID string) (int32, error)
	GetPreferences(ctx context.Context, userID string) ([]*domain.NotificationPreference, error)
	UpsertPreference(ctx context.Context, p *domain.NotificationPreference) error
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Save(ctx context.Context, n *domain.Notification) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error) {
	var n domain.Notification
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

// List returns a page of in-app notifications ordered newest-first.
// cursor is the created_at of the last item from the previous page (RFC3339Nano).
func (r *notificationRepository) List(ctx context.Context, recipientID string, pageSize int, cursor string, unreadOnly bool) ([]*domain.Notification, string, error) {
	if pageSize <= 0 {
		pageSize = 20
	}

	q := r.db.WithContext(ctx).
		Where("recipient_id = ?", recipientID).
		Order("created_at DESC")

	if unreadOnly {
		q = q.Where("read = false")
	}

	if cursor != "" {
		t, err := time.Parse(time.RFC3339Nano, cursor)
		if err == nil {
			q = q.Where("created_at < ?", t)
		}
	}

	// Fetch one extra to detect whether a next page exists.
	var rows []*domain.Notification
	if err := q.Limit(pageSize + 1).Find(&rows).Error; err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(rows) > pageSize {
		nextCursor = rows[pageSize-1].CreatedAt.UTC().Format(time.RFC3339Nano)
		rows = rows[:pageSize]
	}
	return rows, nextCursor, nil
}

// MarkRead marks notifications as read. Pass empty ids to mark everything read.
func (r *notificationRepository) MarkRead(ctx context.Context, recipientID string, ids []string) (int32, error) {
	q := r.db.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("recipient_id = ?", recipientID)
	if len(ids) > 0 {
		q = q.Where("id IN ?", ids)
	}
	result := q.Update("read", true)
	return int32(result.RowsAffected), result.Error
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, recipientID string) (int32, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("recipient_id = ? AND read = false", recipientID).
		Count(&count).Error
	return int32(count), err
}

func (r *notificationRepository) GetPreferences(ctx context.Context, userID string) ([]*domain.NotificationPreference, error) {
	var prefs []*domain.NotificationPreference
	return prefs, r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&prefs).Error
}

func (r *notificationRepository) UpsertPreference(ctx context.Context, p *domain.NotificationPreference) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).
		Where(domain.NotificationPreference{UserID: p.UserID, Category: p.Category}).
		Assign(domain.NotificationPreference{Channels: p.Channels, ID: p.ID}).
		FirstOrCreate(p).Error
}
