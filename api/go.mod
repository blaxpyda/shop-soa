module thugcorp.io/grocery/api

go 1.26.4

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	google.golang.org/grpc v1.81.1
	thugcorp.io/catalog v0.0.0-00010101000000-000000000000
	thugcorp.io/grocery/auth v0.0.0-00010101000000-000000000000
	thugcorp.io/grocery/business v0.0.0-00010101000000-000000000000
	thugcorp.io/grocery/notifications v0.0.0-00010101000000-000000000000
	thugcorp.io/ordering v0.0.0-00010101000000-000000000000
	thugcorp.io/payment v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	thugcorp.io/catalog => ../catalog
	thugcorp.io/grocery/auth => ../auth
	thugcorp.io/grocery/business => ../business
	thugcorp.io/grocery/notifications => ../notifications
	thugcorp.io/ordering => ../ordering
	thugcorp.io/payment => ../payment
)
