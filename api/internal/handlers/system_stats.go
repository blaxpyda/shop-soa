package handlers

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	authpb "thugcorp.io/grocery/auth/proto"
	notificationspb "thugcorp.io/grocery/notifications/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
	catalogpb "thugcorp.io/catalog/proto"
	orderingpb "thugcorp.io/ordering/proto"
	paymentpb "thugcorp.io/payment/proto"
)

type systemService struct {
	Name        string `json:"name"`
	Operational bool   `json:"operational"`
}

type systemStatsResp struct {
	TotalUsers      int32             `json:"total_users"`
	TotalBusinesses int32             `json:"total_businesses"`
	TotalOrders     int               `json:"total_orders"`
	RevenueCents    int64             `json:"revenue_cents"`
	Currency        string            `json:"currency"`
	WeekActivity    [7]float64        `json:"week_activity"`
	Services        []systemService   `json:"services"`
}

// GET /v1/system/stats — super-admin only
func (h *Handlers) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	_, role, _, _ := middleware.ClaimsFromCtx(r.Context())
	if role != "super-admin" {
		respond.Error(w, http.StatusForbidden, "super-admin access required")
		return
	}

	ctx := h.outgoingCtx(r)

	// ── User counts ───────────────────────────────────────────────────────────
	totalUsers := int32(0)
	if ur, err := h.svc.Auth.ListUsers(ctx, &authpb.ListUsersRequest{Page: 1, PageSize: 1}); err == nil {
		totalUsers = ur.Total
	}

	totalBusinesses := int32(0)
	if ar, err := h.svc.Auth.ListUsers(ctx, &authpb.ListUsersRequest{Page: 1, PageSize: 1, Role: "admin"}); err == nil {
		totalBusinesses = ar.Total
	}

	// ── Orders: all-time totals + current-week activity ───────────────────────
	now := time.Now().UTC()
	wd := int(now.Weekday()) // 0 = Sunday
	if wd == 0 {
		wd = 7 // ISO: Mon=1…Sun=7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(wd - 1))
	weekEnd := weekStart.AddDate(0, 0, 7)

	var totalOrders int
	var revenueCents int64
	currency := "GHS"
	var weekCounts [7]int

	if or_, err := h.svc.Ordering.ListOrders(ctx, &orderingpb.ListOrdersRequest{PageSize: 1000}); err == nil {
		for _, o := range or_.Orders {
			if o.Status == orderingpb.OrderStatus_ORDER_STATUS_CANCELLED {
				continue
			}
			totalOrders++
			revenueCents += o.Total
			if o.Currency != "" {
				currency = o.Currency
			}
			if o.CreatedAt != nil {
				ts := o.CreatedAt.AsTime().UTC()
				if !ts.Before(weekStart) && ts.Before(weekEnd) {
					if idx := int(ts.Sub(weekStart).Hours() / 24); idx >= 0 && idx < 7 {
						weekCounts[idx]++
					}
				}
			}
		}
	}

	// Normalise week activity to [0, 1]; always at least 1 as denominator
	maxCount := 1
	for _, c := range weekCounts {
		if c > maxCount {
			maxCount = c
		}
	}
	var weekActivity [7]float64
	for i, c := range weekCounts {
		weekActivity[i] = float64(c) / float64(maxCount)
	}

	// ── Service health checks (2 s timeout each) ──────────────────────────────
	check := func(name string, fn func(context.Context) error) systemService {
		hctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		err := fn(hctx)
		operational := true
		if err != nil {
			c := status.Code(err)
			if c == codes.Unavailable || c == codes.DeadlineExceeded {
				operational = false
			}
		}
		return systemService{Name: name, Operational: operational}
	}

	services := []systemService{
		check("Auth Service", func(ctx context.Context) error {
			_, err := h.svc.Auth.ListUsers(ctx, &authpb.ListUsersRequest{Page: 1, PageSize: 1})
			return err
		}),
		check("Catalog Service", func(ctx context.Context) error {
			_, err := h.svc.Catalog.ListProducts(ctx, &catalogpb.ListProductsRequest{PageSize: 1})
			return err
		}),
		check("Ordering Service", func(ctx context.Context) error {
			_, err := h.svc.Ordering.ListOrders(ctx, &orderingpb.ListOrdersRequest{PageSize: 1})
			return err
		}),
		check("Payment Service", func(ctx context.Context) error {
			_, err := h.svc.Payment.ListTransactions(ctx, &paymentpb.ListTransactionsRequest{PageSize: 1})
			return err
		}),
		check("Notifications", func(ctx context.Context) error {
			_, err := h.svc.Notifications.ListNotifications(ctx, &notificationspb.ListNotificationsRequest{PageSize: 1})
			return err
		}),
	}

	respond.JSON(w, http.StatusOK, systemStatsResp{
		TotalUsers:      totalUsers,
		TotalBusinesses: totalBusinesses,
		TotalOrders:     totalOrders,
		RevenueCents:    revenueCents,
		Currency:        currency,
		WeekActivity:    weekActivity,
		Services:        services,
	})
}
