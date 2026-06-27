# ==============================================================================
# Build stage
# ==============================================================================
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go modules
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/user/go.mod apps/services/user/go.sum ./apps/services/user/

# Download dependencies
RUN go mod download

# Copy source code
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/user/ ./apps/services/user/

# Build
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME} -s -w" \
    -o /app/user-service \
    ./apps/services/user/cmd/user/main.go

# ==============================================================================
# Runtime stage
# ==============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -g '' appuser

COPY --from=builder /app/user-service /app/user-service

USER appuser

EXPOSE 8081

ENTRYPOINT ["/app/user-service"]
