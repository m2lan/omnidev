module github.com/omnidev/gateway

go 1.22.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/omnidev/go-common v0.0.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/lib/pq v1.12.3 // indirect
)

replace github.com/omnidev/go-common => ../../packages/go-common
