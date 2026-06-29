FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/notification/go.mod apps/services/notification/go.sum ./apps/services/notification/
RUN go mod download
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/notification/ ./apps/services/notification/
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/notification-service ./apps/services/notification/cmd/notification/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
COPY --from=builder /app/notification-service /app/notification-service
USER appuser
EXPOSE 8089
ENTRYPOINT ["/app/notification-service"]
