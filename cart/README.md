# Cart Service

A gRPC microservice responsible for managing per-user shopping carts within the grocery SOA platform. Carts are scoped to a `(user_id, business_id)` pair, meaning a user has an independent cart at each store.

---

## Architecture Overview

```
gRPC Client (e.g. API Gateway / Orders Service)
        │
        │  Bearer <JWT>  (gRPC metadata)
        ▼
┌─────────────────────────────────────────────┐
│                gRPC Server                  │
│                                             │
│  ┌──────────────────────────────────────┐   │
│  │         Interceptor Chain            │   │
│  │  1. LoggingInterceptor               │   │
│  │  2. RecoveryInterceptor              │   │
│  │  3. AuthInterceptor (JWT → userID)   │   │
│  └──────────────────┬───────────────────┘   │
│                     │                       │
│  ┌──────────────────▼───────────────────┐   │
│  │           CartHandler                │   │
│  │  GetCart / AddToCart /               │   │
│  │  RemoveFromCart / ClearCart          │   │
│  └──────────────────┬───────────────────┘   │
│                     │                       │
│  ┌──────────────────▼───────────────────┐   │
│  │           CartService                │   │
│  │  Business logic & validation         │   │
│  └──────────────────┬───────────────────┘   │
│                     │                       │
│  ┌──────────────────▼───────────────────┐   │
│  │         CartRepository               │   │
│  │  Redis read / write (JSON)           │   │
│  └──────────────────┬───────────────────┘   │
└─────────────────────┼───────────────────────┘
                      │
                      ▼
                  Redis
            key: "cart:<userID>"
            TTL: 7 days
```

---

## Components

### `main.go`

Entry point. Loads config, wires dependencies, chains interceptors, and starts the gRPC server on the configured port (default `50052`).

### `config/`

Loads configuration via [Viper](https://github.com/spf13/viper) from a YAML file (`config/config.yaml` or `config/local.yaml`). Environment variables prefixed with `CART_` override file values.

| Section | Key fields |
|---------|-----------|
| `app`   | `name`, `environment`, `port` |
| `grpc`  | `port` (gRPC listener, default `50052`) |
| `redis` | `addr`, `password`, `db`, `pool_size` |
| `jwt`   | `secret`, `access_token_ttl` |

See [config/local.example.yaml](config/local.example.yaml) for a full template.

### `proto/`

Defines the `CartService` contract. Generated Go files (`cart.pb.go`, `cart_grpc.pb.go`) are committed alongside `cart.proto`.

**RPC methods:**

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `GetCart` | `GetCartRequest` | `Cart` | Fetch the caller's current cart |
| `AddToCart` | `AddToCartRequest` | `Cart` | Add or increment a product; returns updated cart |
| `RemoveFromCart` | `RemoveFromCartRequest` | `Cart` | Remove a product line; returns updated cart |
| `ClearCart` | `ClearCartRequest` | `EmptyResponse` | Delete the entire cart |

### `internal/middleware/`

Three gRPC unary interceptors applied in order:

| Interceptor | File | Responsibility |
|-------------|------|----------------|
| `LoggingInterceptor` | `logging.go` | Logs method name, duration, and gRPC status code for every call |
| `RecoveryInterceptor` | `recovery.go` | Recovers from panics and returns `codes.Internal` instead of crashing |
| `AuthInterceptor` | `auth.go` | Validates the `Authorization: Bearer <token>` header; injects `userID` and `businessID` into context |

The auth interceptor parses a JWT signed with `HS256`. Claims must contain `user_id` (and optionally `business_id`). The extracted `userID` is made available to handlers via `middleware.UserIDKey` on the context.

### `internal/handlers.go`

`CartHandler` implements the generated `CartServiceServer` interface. Each method:

1. Pulls `userID` from context (set by `AuthInterceptor`).
2. Delegates to `CartService`.
3. Maps the domain `Cart` struct to the protobuf `Cart` message.

### `internal/services.go`

`CartService` contains the business logic layer:

- Validates that `quantity > 0` before adding items.
- Delegates all persistence to `CartRepository`.
- After any mutation, re-fetches the cart and returns the updated state.

### `internal/repository.go`

`CartRepository` persists carts in Redis as JSON blobs under the key `cart:<userID>`.

- `GetCart` — returns an empty cart (no error) when no key exists.
- `AddItem` — read-modify-write: increments quantity if the product already exists, otherwise appends a new `CartItem`.
- `RemoveItem` — filters out the matching product.
- `ClearCart` — deletes the key entirely.
- All writes set a **7-day TTL**, so abandoned carts expire automatically.

### `internal/domain/cart.go`

Plain Go structs (`Cart`, `CartItem`) representing the in-memory and serialised cart shape. GORM tags are present for a potential future SQL-backed implementation but are unused while the repository is Redis-only.

---

## Request Flow (AddToCart example)

```
Client sends AddToCartRequest + JWT in metadata
    │
    ▼
LoggingInterceptor (records start time)
    │
    ▼
RecoveryInterceptor (deferred panic guard)
    │
    ▼
AuthInterceptor → validates JWT → writes userID to ctx
    │
    ▼
CartHandler.AddToCart
  └─ reads userID from ctx
  └─ calls CartService.AddToCart(ctx, userID, productID, quantity)
        └─ validates quantity > 0
        └─ calls CartRepository.AddItem(ctx, userID, productID, quantity)
              └─ GET cart:<userID> from Redis
              └─ upsert item in Items slice
              └─ SET cart:<userID> (JSON, TTL 7d)
        └─ calls CartRepository.GetCart → returns updated Cart
  └─ maps domain.Cart → pb.Cart
    │
    ▼
LoggingInterceptor logs duration + status code
    │
    ▼
Response returned to client
```

---

## Configuration

Copy the example and fill in your values:

```bash
cp config/local.example.yaml config/config.yaml
```

Key values to change for production:

- `jwt.secret` — use a strong random secret, never the placeholder.
- `redis.password` — set a real password.
- `app.environment` — set to `production`.

---

## Running

```bash
go run ./main.go
```

The server listens on the gRPC port defined in config (default `50052`).

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `google.golang.org/grpc` | gRPC server and interceptors |
| `google.golang.org/protobuf` | Protobuf runtime |
| `github.com/go-redis/redis/v8` | Redis client |
| `github.com/golang-jwt/jwt/v5` | JWT parsing and validation |
| `github.com/spf13/viper` | Config loading (YAML + env vars) |
| `gorm.io/gorm` | ORM structs (domain layer, unused at runtime) |
