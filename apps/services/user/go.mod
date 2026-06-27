module github.com/omnidev/services/user

go 1.22.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/go-playground/validator/v10 v10.20.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/omnidev/go-common v0.0.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.24.0
	golang.org/x/oauth2 v0.21.0
)

replace github.com/omnidev/go-common => ../../../packages/go-common
