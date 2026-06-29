FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/billing/go.mod apps/services/billing/go.sum ./apps/services/billing/
RUN go mod download
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/billing/ ./apps/services/billing/
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/billing-service ./apps/services/billing/cmd/billing/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
COPY --from=builder /app/billing-service /app/billing-service
USER appuser
EXPOSE 8088
ENTRYPOINT ["/app/billing-service"]
