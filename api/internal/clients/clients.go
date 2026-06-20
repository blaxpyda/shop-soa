package clients

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "thugcorp.io/grocery/auth/proto"
	businesspb "thugcorp.io/grocery/business/proto"
	cartpb "thugcorp.io/grocery/cart/proto"
	inventorypb "thugcorp.io/grocery/inventory/proto"
	notificationspb "thugcorp.io/grocery/notifications/proto"
	orderpb "thugcorp.io/grocery/order/proto"
	paymentpb "thugcorp.io/grocery/payment/proto"
	productpb "thugcorp.io/grocery/product/proto"
	transactionpb "thugcorp.io/grocery/transaction/proto"
)

// Targets holds the gRPC address for each downstream service.
// Values come from environment variables with sensible localhost defaults.
type Targets struct {
	Auth          string
	Products      string
	Cart          string
	Orders        string
	Business      string
	Inventory     string
	Notifications string
	Transactions  string
	Payments      string
}

// Services wraps every typed gRPC client and the raw connections so they can
// all be closed together.
type Services struct {
	Auth          authpb.AuthServiceClient
	Products      productpb.ProductServiceClient
	Cart          cartpb.CartServiceClient
	Orders        orderpb.OrderServiceClient
	Business      businesspb.BusinessServiceClient
	Inventory     inventorypb.InventoryServiceClient
	Notifications notificationspb.NotificationServiceClient
	Transactions  transactionpb.TransactionServiceClient
	Payment       paymentpb.PaymentServiceClient

	conns []*grpc.ClientConn
}

// Close drains all open gRPC connections. Call defer svc.Close() in main.
func (s *Services) Close() {
	for _, c := range s.conns {
		c.Close()
	}
}

// Dial opens one insecure gRPC connection per service and returns the typed
// client bundle. Swap insecure.NewCredentials() for TLS creds in production.
func Dial(t Targets) (*Services, error) {
	dial := func(addr string) (*grpc.ClientConn, error) {
		return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	authConn, err := dial(t.Auth)
	if err != nil {
		return nil, err
	}
	productConn, err := dial(t.Products)
	if err != nil {
		return nil, err
	}
	cartConn, err := dial(t.Cart)
	if err != nil {
		return nil, err
	}
	orderConn, err := dial(t.Orders)
	if err != nil {
		return nil, err
	}
	businessConn, err := dial(t.Business)
	if err != nil {
		return nil, err
	}
	inventoryConn, err := dial(t.Inventory)
	if err != nil {
		return nil, err
	}
	notificationsConn, err := dial(t.Notifications)
	if err != nil {
		return nil, err
	}
	transactionConn, err := dial(t.Transactions)
	if err != nil {
		return nil, err
	}
	paymentConn, err := dial(t.Payments)
	if err != nil {
		return nil, err
	}

	return &Services{
		Auth:          authpb.NewAuthServiceClient(authConn),
		Products:      productpb.NewProductServiceClient(productConn),
		Cart:          cartpb.NewCartServiceClient(cartConn),
		Orders:        orderpb.NewOrderServiceClient(orderConn),
		Business:      businesspb.NewBusinessServiceClient(businessConn),
		Inventory:     inventorypb.NewInventoryServiceClient(inventoryConn),
		Notifications: notificationspb.NewNotificationServiceClient(notificationsConn),
		Transactions:  transactionpb.NewTransactionServiceClient(transactionConn),
		Payment:       paymentpb.NewPaymentServiceClient(paymentConn),
		conns: []*grpc.ClientConn{
			authConn, productConn, cartConn, orderConn,
			businessConn, inventoryConn, notificationsConn, transactionConn, paymentConn,
		},
	}, nil
}
