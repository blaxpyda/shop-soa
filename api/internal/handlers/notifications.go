package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	notificationspb "thugcorp.io/grocery/notifications/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/notifications
func (h *Handlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	resp, err := h.svc.Notifications.ListNotifications(h.outgoingCtx(r), &notificationspb.ListNotificationsRequest{
		UserId:     userID,
		PageSize:   int32(pageSize),
		PageToken:  r.URL.Query().Get("page_token"),
		UnreadOnly: unreadOnly,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// PUT /v1/notifications/read
func (h *Handlers) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		NotificationIDs []string `json:"notification_ids"` // empty = mark all
	}
	json.NewDecoder(r.Body).Decode(&body)

	resp, err := h.svc.Notifications.MarkRead(h.outgoingCtx(r), &notificationspb.MarkReadRequest{
		UserId:          userID,
		NotificationIds: body.NotificationIDs,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/notifications/unread-count
func (h *Handlers) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Notifications.GetUnreadCount(h.outgoingCtx(r), &notificationspb.GetUnreadCountRequest{
		UserId: userID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/notifications/preferences
func (h *Handlers) GetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Notifications.GetPreferences(h.outgoingCtx(r), &notificationspb.GetPreferencesRequest{
		UserId: userID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// PUT /v1/notifications/preferences
func (h *Handlers) UpdateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		Preferences []struct {
			Category        string   `json:"category"`
			EnabledChannels []string `json:"enabled_channels"`
		} `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	prefs := make([]*notificationspb.ChannelPreference, 0, len(body.Preferences))
	for _, p := range body.Preferences {
		channels := make([]notificationspb.Channel, 0, len(p.EnabledChannels))
		for _, ch := range p.EnabledChannels {
			if v, ok := notificationspb.Channel_value[ch]; ok {
				channels = append(channels, notificationspb.Channel(v))
			}
		}
		prefs = append(prefs, &notificationspb.ChannelPreference{
			Category:        notificationspb.Category(notificationspb.Category_value[p.Category]),
			EnabledChannels: channels,
		})
	}

	resp, err := h.svc.Notifications.UpdatePreferences(h.outgoingCtx(r), &notificationspb.UpdatePreferencesRequest{
		UserId:      userID,
		Preferences: prefs,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
