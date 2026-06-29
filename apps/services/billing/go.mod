module github.com/omnidev/services/billing

go 1.22.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/omnidev/go-common v0.0.0
	go.uber.org/zap v1.27.0
)

replace github.com/omnidev/go-common => ../../../packages/go-common
