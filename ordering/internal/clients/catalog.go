package clients

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	catalogpb "thugcorp.io/catalog/proto"
)

// CatalogProduct is the ordering service's view of a product — only the
// fields needed to populate a cart item are included.
type CatalogProduct struct {
	ID         string
	BusinessID string
	Name       string
	Price      int64
	Currency   string
	Active     bool
}

// ReserveItem is one line in a reservation request.
type ReserveItem struct {
	ProductID string
	Quantity  int64
}

// ReserveResult is returned by Reserve.
type ReserveResult struct {
	ReservationID string
	Held          bool
	Shortfalls    []string // product IDs with insufficient stock
}

// CatalogClient is the interface ordering uses to talk to the catalog service.
// Defined as an interface so it can be stubbed in tests.
type CatalogClient interface {
	GetProduct(ctx context.Context, productID string) (*CatalogProduct, error)
	Reserve(ctx context.Context, orderID string, items []ReserveItem, idempotencyKey string) (*ReserveResult, error)
	CommitReservation(ctx context.Context, reservationID string) error
	ReleaseReservation(ctx context.Context, reservationID, reason string) error
}

type catalogGRPCClient struct {
	client catalogpb.CatalogServiceClient
}

// NewCatalogClient dials the catalog gRPC service and returns a client and
// the underlying connection (call conn.Close() on shutdown).
func NewCatalogClient(addr string) (CatalogClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial catalog at %s: %w", addr, err)
	}
	return &catalogGRPCClient{client: catalogpb.NewCatalogServiceClient(conn)}, conn, nil
}

func (c *catalogGRPCClient) GetProduct(ctx context.Context, productID string) (*CatalogProduct, error) {
	resp, err := c.client.GetProduct(ctx, &catalogpb.GetProductRequest{ProductId: productID})
	if err != nil {
		return nil, fmt.Errorf("catalog.GetProduct(%s): %w", productID, err)
	}
	return &CatalogProduct{
		ID:         resp.Id,
		BusinessID: resp.BusinessId,
		Name:       resp.Name,
		Price:      resp.Price,
		Currency:   resp.Currency,
		Active:     resp.Active,
	}, nil
}

func (c *catalogGRPCClient) Reserve(ctx context.Context, orderID string, items []ReserveItem, idempotencyKey string) (*ReserveResult, error) {
	pbItems := make([]*catalogpb.ReserveItem, len(items))
	for i, item := range items {
		pbItems[i] = &catalogpb.ReserveItem{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	resp, err := c.client.Reserve(ctx, &catalogpb.ReserveRequest{
		OrderId:        orderID,
		Items:          pbItems,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("catalog.Reserve: %w", err)
	}
	result := &ReserveResult{
		ReservationID: resp.ReservationId,
		Held:          resp.Held,
	}
	for _, sf := range resp.Shortfalls {
		result.Shortfalls = append(result.Shortfalls, sf.ProductId)
	}
	return result, nil
}

func (c *catalogGRPCClient) CommitReservation(ctx context.Context, reservationID string) error {
	_, err := c.client.CommitReservation(ctx, &catalogpb.CommitReservationRequest{
		ReservationId: reservationID,
	})
	if err != nil {
		return fmt.Errorf("catalog.CommitReservation(%s): %w", reservationID, err)
	}
	return nil
}

func (c *catalogGRPCClient) ReleaseReservation(ctx context.Context, reservationID, reason string) error {
	_, err := c.client.ReleaseReservation(ctx, &catalogpb.ReleaseReservationRequest{
		ReservationId: reservationID,
		Reason:        reason,
	})
	if err != nil {
		return fmt.Errorf("catalog.ReleaseReservation(%s): %w", reservationID, err)
	}
	return nil
}
