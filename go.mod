module github.com/Archiit19/customer-service-go

go 1.24.0

require (
github.com/go-chi/chi/v5 v5.2.3
github.com/google/uuid v1.6.0
github.com/jackc/pgx/v5 v5.7.6
github.com/nyaruka/phonenumbers v1.6.6
go.uber.org/zap v0.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.24.0 // indirect
google.golang.org/protobuf v1.36.10 // indirect
)

replace go.uber.org/zap => ./third_party/zap
