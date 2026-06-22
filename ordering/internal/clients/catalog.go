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

// CatalogClient is the interface ordering uses to look up products.
// Defined as an interface so it can be stubbed in tests.
type CatalogClient interface {
	GetProduct(ctx context.Context, productID string) (*CatalogProduct, error)
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
