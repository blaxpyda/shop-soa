package internal

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"thugcorp.io/grocery/notifications/internal/domain"
	"thugcorp.io/grocery/notifications/internal/middleware"
	pb "thugcorp.io/grocery/notifications/proto"
)

type notificationHandler struct {
	pb.UnimplementedNotificationServiceServer
	svc NotificationService
}

func NewNotificationHandler(svc NotificationService) *notificationHandler {
	return &notificationHandler{svc: svc}
}

// ---- Send ----

func (h *notificationHandler) Send(ctx context.Context, req *pb.SendRequest) (*pb.SendResponse, error) {
	if req.Recipient == nil {
		return nil, status.Error(codes.InvalidArgument, "recipient is required")
	}

	input := domain.SendInput{
		Email:          req.Recipient.GetEmail(),
		Phone:          req.Recipient.GetPhone(),
		PushToken:      req.Recipient.GetPushToken(),
		TemplateID:     req.TemplateId,
		Variables:      req.Variables,
		Category:       req.Category.String(),
		IdempotencyKey: req.IdempotencyKey,
	}

	switch id := req.Recipient.Id.(type) {
	case *pb.Recipient_UserId:
		input.RecipientUserID = id.UserId
	case *pb.Recipient_BusinessId:
		input.RecipientBusinessID = id.BusinessId
	}

	for _, ch := range req.Channels {
		input.Channels = append(input.Channels, ch.String())
	}

	notifID, results, err := h.svc.Send(ctx, input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	pbResults := make([]*pb.ChannelResult, 0, len(results))
	for _, r := range results {
		pbResults = append(pbResults, &pb.ChannelResult{
			Channel: pb.Channel(pb.Channel_value[r.Channel]),
			Status:  pb.DeliveryStatus(pb.DeliveryStatus_value[r.Status]),
			Detail:  r.Detail,
		})
	}

	return &pb.SendResponse{
		NotificationId: notifID,
		Results:        pbResults,
	}, nil
}

// ---- Inbox ----

func (h *notificationHandler) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	callerID, err := callerIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	rows, nextCursor, err := h.svc.ListNotifications(ctx, callerID, int(req.PageSize), req.PageToken, req.UnreadOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	pbItems := make([]*pb.Notification, 0, len(rows))
	for _, n := range rows {
		pbItems = append(pbItems, mapNotificationToProto(n))
	}

	return &pb.ListNotificationsResponse{
		Notifications: pbItems,
		NextPageToken: nextCursor,
	}, nil
}

func (h *notificationHandler) MarkRead(ctx context.Context, req *pb.MarkReadRequest) (*pb.MarkReadResponse, error) {
	callerID, err := callerIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	count, err := h.svc.MarkRead(ctx, callerID, req.NotificationIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.MarkReadResponse{UpdatedCount: count}, nil
}

func (h *notificationHandler) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountResponse, error) {
	callerID, err := callerIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	count, err := h.svc.GetUnreadCount(ctx, callerID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.GetUnreadCountResponse{Count: count}, nil
}

// ---- Preferences ----

func (h *notificationHandler) GetPreferences(ctx context.Context, req *pb.GetPreferencesRequest) (*pb.NotificationPreferences, error) {
	callerID, err := callerIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	prefs, err := h.svc.GetPreferences(ctx, callerID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapPreferencesToProto(callerID, prefs), nil
}

func (h *notificationHandler) UpdatePreferences(ctx context.Context, req *pb.UpdatePreferencesRequest) (*pb.NotificationPreferences, error) {
	callerID, err := callerIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	domainPrefs := make([]*domain.NotificationPreference, 0, len(req.Preferences))
	for _, cp := range req.Preferences {
		chs := make([]string, 0, len(cp.EnabledChannels))
		for _, ch := range cp.EnabledChannels {
			chs = append(chs, ch.String())
		}
		chJSON, _ := json.Marshal(chs)
		domainPrefs = append(domainPrefs, &domain.NotificationPreference{
			UserID:   callerID,
			Category: cp.Category.String(),
			Channels: string(chJSON),
		})
	}

	updated, err := h.svc.UpdatePreferences(ctx, callerID, domainPrefs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapPreferencesToProto(callerID, updated), nil
}

// ---- Helpers ----

func callerIDFromCtx(ctx context.Context) (string, error) {
	id, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || id == "" {
		return "", status.Error(codes.Unauthenticated, "user ID not found in context")
	}
	return id, nil
}

func mapNotificationToProto(n *domain.Notification) *pb.Notification {
	var data map[string]string
	if n.Data != "" {
		json.Unmarshal([]byte(n.Data), &data)
	}
	return &pb.Notification{
		Id:        n.ID,
		Title:     n.Title,
		Body:      n.Body,
		Category:  pb.Category(pb.Category_value[n.Category]),
		Read:      n.Read,
		CreatedAt: timestamppb.New(n.CreatedAt),
		Data:      data,
	}
}

func mapPreferencesToProto(userID string, prefs []*domain.NotificationPreference) *pb.NotificationPreferences {
	cpList := make([]*pb.ChannelPreference, 0, len(prefs))
	for _, p := range prefs {
		var chStrings []string
		json.Unmarshal([]byte(p.Channels), &chStrings)

		chs := make([]pb.Channel, 0, len(chStrings))
		for _, cs := range chStrings {
			if v, ok := pb.Channel_value[cs]; ok {
				chs = append(chs, pb.Channel(v))
			}
		}
		cpList = append(cpList, &pb.ChannelPreference{
			Category:       pb.Category(pb.Category_value[p.Category]),
			EnabledChannels: chs,
		})
	}
	return &pb.NotificationPreferences{
		UserId:      userID,
		Preferences: cpList,
	}
}
