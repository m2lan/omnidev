# ==============================================================================
# Build stage
# ==============================================================================
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go modules
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/gateway/go.mod apps/gateway/go.sum ./apps/gateway/

# Download dependencies
RUN go mod download

# Copy source code
COPY packages/go-common/ ./packages/go-common/
COPY apps/gateway/ ./apps/gateway/

# Build
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME} -s -w" \
    -o /app/gateway \
    ./apps/gateway/cmd/gateway/main.go

# ==============================================================================
# Runtime stage
# ==============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -g '' appuser

COPY --from=builder /app/gateway /app/gateway

USER appuser

EXPOSE 9090

ENTRYPOINT ["/app/gateway"]
