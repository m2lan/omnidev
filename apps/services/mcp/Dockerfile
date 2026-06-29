FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/mcp/go.mod apps/services/mcp/go.sum ./apps/services/mcp/
RUN go mod download
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/mcp/ ./apps/services/mcp/
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/mcp-service ./apps/services/mcp/cmd/mcp/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata docker-cli
RUN adduser -D -g '' appuser
COPY --from=builder /app/mcp-service /app/mcp-service
USER appuser
EXPOSE 8085
ENTRYPOINT ["/app/mcp-service"]
