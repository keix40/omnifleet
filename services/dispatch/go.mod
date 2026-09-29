module github.com/keix40/omnifleet/services/dispatch

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/keix40/omnifleet/gen/go v0.0.0
	github.com/keix40/omnifleet/pkg v0.0.0
	google.golang.org/grpc v1.69.4
)

replace (
	github.com/keix40/omnifleet/gen/go => ../../gen/go
	github.com/keix40/omnifleet/pkg => ../../pkg
)
