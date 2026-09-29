module github.com/keix40/omnifleet/services/gateway

go 1.22

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-chi/cors v1.2.1
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.7.2
	github.com/keix40/omnifleet/gen/go v0.0.0
	github.com/keix40/omnifleet/pkg v0.0.0
	github.com/nats-io/nats.go v1.38.0
	google.golang.org/grpc v1.69.4
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/protobuf v1.36.2 // indirect
)

replace (
	github.com/keix40/omnifleet/gen/go => ../../gen/go
	github.com/keix40/omnifleet/pkg => ../../pkg
)
