package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	orderingpb "thugcorp.io/ordering/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/my/stats — caller's own today-stats + recent orders (user_id from JWT)
func (h *Handlers) GetMyStats(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	orders, err := h.svc.Ordering.ListOrders(h.outgoingCtx(r), &orderingpb.ListOrdersRequest{
		Filter:   &orderingpb.ListOrdersRequest_UserId{UserId: userID},
		PageSize: 200,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}

	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayEnd := todayStart.Add(24 * time.Hour)

	type recentOrder struct {
		OrderNo    string `json:"order_no"`
		ItemCount  int    `json:"item_count"`
		Status     string `json:"status"`
		TotalCents int64  `json:"total_cents"`
	}

	var revCents int64
	var orderCount int
	var itemCount int64
	recent := make([]recentOrder, 0, 5)

	for _, o := range orders.Orders {
		if o.Status == orderingpb.OrderStatus_ORDER_STATUS_CANCELLED {
			continue
		}
		var ts time.Time
		if o.CreatedAt != nil {
			ts = o.CreatedAt.AsTime().UTC()
		}

		if !ts.Before(todayStart) && ts.Before(todayEnd) {
			revCents += o.Total
			orderCount++
			for _, item := range o.Items {
				itemCount += item.Quantity
			}
		}

		if len(recent) < 5 {
			ic := 0
			for _, item := range o.Items {
				ic += int(item.Quantity)
			}
			id := o.Id
			if len(id) > 8 {
				id = id[:8]
			}
			recent = append(recent, recentOrder{
				OrderNo:    "#" + id,
				ItemCount:  ic,
				Status:     orderStatusLabel(o.Status),
				TotalCents: o.Total,
			})
		}
	}

	var avgBasket int64
	if orderCount > 0 {
		avgBasket = revCents / int64(orderCount)
	}

	type myStatsResp struct {
		RevenueCents   int64         `json:"revenue_cents"`
		OrderCount     int           `json:"order_count"`
		ItemCount      int64         `json:"item_count"`
		AvgBasketCents int64         `json:"avg_basket_cents"`
		RecentOrders   []recentOrder `json:"recent_orders"`
	}

	respond.JSON(w, http.StatusOK, myStatsResp{
		RevenueCents:   revCents,
		OrderCount:     orderCount,
		ItemCount:      itemCount,
		AvgBasketCents: avgBasket,
		RecentOrders:   recent,
	})
}

func orderStatusLabel(s orderingpb.OrderStatus) string {
	switch s {
	case orderingpb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT:
		return "Pending"
	case orderingpb.OrderStatus_ORDER_STATUS_CONFIRMED:
		return "Confirmed"
	case orderingpb.OrderStatus_ORDER_STATUS_PREPARING:
		return "Preparing"
	case orderingpb.OrderStatus_ORDER_STATUS_OUT_FOR_DELIVERY:
		return "Delivering"
	case orderingpb.OrderStatus_ORDER_STATUS_DELIVERED:
		return "Delivered"
	default:
		return "Processing"
	}
}

type statsBar struct {
	Label      string `json:"label"`
	ValueCents int64  `json:"value_cents"`
}

type statsTopProduct struct {
	Name         string  `json:"name"`
	RevenueCents int64   `json:"revenue_cents"`
	Units        int64   `json:"units"`
	Pct          float64 `json:"pct"`
}

type statsResp struct {
	RevenueCents  int64             `json:"revenue_cents"`
	RevenueChange float64           `json:"revenue_change"`
	OrderCount    int               `json:"order_count"`
	AvgOrderCents int64             `json:"avg_order_cents"`
	Bars          []statsBar        `json:"bars"`
	TopProducts   []statsTopProduct `json:"top_products"`
}

// GET /v1/stats?range=today|week|month|year
func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	_, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())
	if businessID == "" {
		respond.Error(w, http.StatusBadRequest, "no business associated with this account")
		return
	}

	orders, err := h.svc.Ordering.ListOrders(h.outgoingCtx(r), &orderingpb.ListOrdersRequest{
		Filter:   &orderingpb.ListOrdersRequest_BusinessId{BusinessId: businessID},
		PageSize: 1000,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, buildStats(orders.Orders, r.URL.Query().Get("range"), time.Now().UTC()))
}

func buildStats(orders []*orderingpb.Order, rangeParam string, now time.Time) statsResp {
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var start, end, prevStart, prevEnd time.Time
	var bars []statsBar

	switch rangeParam {
	case "today":
		start = midnight
		end = start.Add(24 * time.Hour)
		prevStart = start.AddDate(0, 0, -1)
		prevEnd = start
		bars = todayBars()
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 1, 0)
		prevStart = start.AddDate(0, -1, 0)
		prevEnd = start
		bars = weekBarsInMonth()
	case "year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(1, 0, 0)
		prevStart = start.AddDate(-1, 0, 0)
		prevEnd = start
		bars = monthBars()
	default: // "week"
		wd := int(now.Weekday())
		if wd == 0 {
			wd = 7 // ISO: Mon=1…Sun=7
		}
		start = midnight.AddDate(0, 0, -(wd - 1))
		end = start.AddDate(0, 0, 7)
		prevStart = start.AddDate(0, 0, -7)
		prevEnd = start
		bars = weekdayBars()
	}

	barTotals := make([]int64, len(bars))
	productMap := map[string]*statsTopProduct{}
	var revCents, prevRevCents int64
	var orderCount int

	for _, o := range orders {
		if o.Status == orderingpb.OrderStatus_ORDER_STATUS_CANCELLED {
			continue
		}
		var ts time.Time
		if o.CreatedAt != nil {
			ts = o.CreatedAt.AsTime().UTC()
		}

		switch {
		case !ts.Before(start) && ts.Before(end):
			revCents += o.Total
			orderCount++
			if idx := bucketIndex(ts, rangeParam, start); idx >= 0 && idx < len(bars) {
				barTotals[idx] += o.Total
			}
			for _, item := range o.Items {
				p := productMap[item.ProductName]
				if p == nil {
					productMap[item.ProductName] = &statsTopProduct{Name: item.ProductName}
					p = productMap[item.ProductName]
				}
				p.RevenueCents += item.LineTotal
				p.Units += item.Quantity
			}
		case !ts.Before(prevStart) && ts.Before(prevEnd):
			prevRevCents += o.Total
		}
	}

	for i := range bars {
		bars[i].ValueCents = barTotals[i]
	}

	var revChange float64
	if prevRevCents > 0 {
		revChange = float64(revCents-prevRevCents) / float64(prevRevCents)
	} else if revCents > 0 {
		revChange = 1.0
	}

	var avgOrder int64
	if orderCount > 0 {
		avgOrder = revCents / int64(orderCount)
	}

	// Top 5 products by revenue
	products := make([]*statsTopProduct, 0, len(productMap))
	for _, p := range productMap {
		products = append(products, p)
	}
	sort.Slice(products, func(i, j int) bool {
		return products[i].RevenueCents > products[j].RevenueCents
	})
	if len(products) > 5 {
		products = products[:5]
	}
	top := make([]statsTopProduct, 0, len(products))
	var maxRev int64
	if len(products) > 0 {
		maxRev = products[0].RevenueCents
	}
	for _, p := range products {
		var pct float64
		if maxRev > 0 {
			pct = float64(p.RevenueCents) / float64(maxRev)
		}
		top = append(top, statsTopProduct{Name: p.Name, RevenueCents: p.RevenueCents, Units: p.Units, Pct: pct})
	}

	return statsResp{
		RevenueCents:  revCents,
		RevenueChange: revChange,
		OrderCount:    orderCount,
		AvgOrderCents: avgOrder,
		Bars:          bars,
		TopProducts:   top,
	}
}

// bucketIndex returns which bar slot ts falls into for the given range.
func bucketIndex(ts time.Time, rangeParam string, start time.Time) int {
	switch rangeParam {
	case "today":
		// 7 two-hour bars starting at 6 am
		diff := ts.Sub(start.Add(6 * time.Hour))
		if diff < 0 || diff >= 14*time.Hour {
			return -1
		}
		return int(diff.Hours() / 2)
	case "week":
		return int(ts.Sub(start).Hours() / 24)
	case "month":
		return int(ts.Sub(start).Hours()/24) / 7
	case "year":
		return int(ts.Month()) - 1
	default:
		return -1
	}
}

func todayBars() []statsBar {
	return []statsBar{
		{Label: "6a"}, {Label: "8a"}, {Label: "10a"},
		{Label: "12p"}, {Label: "2p"}, {Label: "4p"}, {Label: "6p"},
	}
}

func weekdayBars() []statsBar {
	return []statsBar{
		{Label: "Mon"}, {Label: "Tue"}, {Label: "Wed"},
		{Label: "Thu"}, {Label: "Fri"}, {Label: "Sat"}, {Label: "Sun"},
	}
}

func weekBarsInMonth() []statsBar {
	bars := make([]statsBar, 5)
	for i := range bars {
		bars[i] = statsBar{Label: fmt.Sprintf("W%d", i+1)}
	}
	return bars
}

func monthBars() []statsBar {
	names := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	bars := make([]statsBar, 12)
	for i, n := range names {
		bars[i] = statsBar{Label: n}
	}
	return bars
}
