# ==============================================================================
# Build stage
#
# Build context is the repository root (the directory containing both
# `omnidev/` and `knowledge-engine/`) so the gateway can import the shared
# knowledge engine module.
# ==============================================================================
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

ENV GOWORK=off

# Copy module files first for better layer caching
COPY omnidev/packages/go-common/go.mod ./omnidev/packages/go-common/
COPY knowledge-engine/go.mod knowledge-engine/go.sum ./knowledge-engine/
COPY omnidev/apps/gateway/go.mod omnidev/apps/gateway/go.sum ./omnidev/apps/gateway/

RUN cd omnidev/apps/gateway && go mod download

# Copy source code
COPY omnidev/packages/go-common/ ./omnidev/packages/go-common/
COPY knowledge-engine/ ./knowledge-engine/
COPY omnidev/apps/gateway/ ./omnidev/apps/gateway/

# Build
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN cd omnidev/apps/gateway && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME} -s -w" \
    -o /app/gateway ./cmd/gateway

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
