package clients

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "thugcorp.io/grocery/auth/proto"
	businesspb "thugcorp.io/grocery/business/proto"
	notificationspb "thugcorp.io/grocery/notifications/proto"
	catalogpb "thugcorp.io/catalog/proto"
	orderingpb "thugcorp.io/ordering/proto"
	paymentpb "thugcorp.io/payment/proto"
)

type Targets struct {
	Auth          string
	Business      string
	Catalog       string
	Ordering      string
	Payment       string
	Notifications string
}

type Services struct {
	Auth          authpb.IdentityServiceClient
	Business      businesspb.BusinessServiceClient
	Catalog       catalogpb.CatalogServiceClient
	Ordering      orderingpb.OrderingServiceClient
	Payment       paymentpb.PaymentServiceClient
	Notifications notificationspb.NotificationServiceClient

	conns []*grpc.ClientConn
}

func (s *Services) Close() {
	for _, c := range s.conns {
		c.Close()
	}
}

func Dial(t Targets) (*Services, error) {
	dial := func(addr string) (*grpc.ClientConn, error) {
		return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	authConn, err := dial(t.Auth)
	if err != nil {
		return nil, err
	}
	businessConn, err := dial(t.Business)
	if err != nil {
		return nil, err
	}
	catalogConn, err := dial(t.Catalog)
	if err != nil {
		return nil, err
	}
	orderingConn, err := dial(t.Ordering)
	if err != nil {
		return nil, err
	}
	paymentConn, err := dial(t.Payment)
	if err != nil {
		return nil, err
	}
	notificationsConn, err := dial(t.Notifications)
	if err != nil {
		return nil, err
	}

	return &Services{
		Auth:          authpb.NewIdentityServiceClient(authConn),
		Business:      businesspb.NewBusinessServiceClient(businessConn),
		Catalog:       catalogpb.NewCatalogServiceClient(catalogConn),
		Ordering:      orderingpb.NewOrderingServiceClient(orderingConn),
		Payment:       paymentpb.NewPaymentServiceClient(paymentConn),
		Notifications: notificationspb.NewNotificationServiceClient(notificationsConn),
		conns: []*grpc.ClientConn{
			authConn, businessConn, catalogConn, orderingConn, paymentConn, notificationsConn,
		},
	}, nil
}
