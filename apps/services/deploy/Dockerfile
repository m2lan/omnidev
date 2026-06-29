FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/deploy/go.mod apps/services/deploy/go.sum ./apps/services/deploy/
RUN go mod download
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/deploy/ ./apps/services/deploy/
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/deploy-service ./apps/services/deploy/cmd/deploy/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata docker-cli
RUN adduser -D -g '' appuser
COPY --from=builder /app/deploy-service /app/deploy-service
USER appuser
EXPOSE 8087
ENTRYPOINT ["/app/deploy-service"]
