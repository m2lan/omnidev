FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/agent/go.mod apps/services/agent/go.sum ./apps/services/agent/
RUN go mod download
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/agent/ ./apps/services/agent/
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/agent-service ./apps/services/agent/cmd/agent/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata python3 nodejs go
RUN adduser -D -g '' appuser
COPY --from=builder /app/agent-service /app/agent-service
USER appuser
EXPOSE 8084
ENTRYPOINT ["/app/agent-service"]
