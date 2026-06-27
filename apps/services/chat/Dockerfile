FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/chat/go.mod apps/services/chat/go.sum ./apps/services/chat/
RUN go mod download
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/chat/ ./apps/services/chat/
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME} -s -w" \
    -o /app/chat-service \
    ./apps/services/chat/cmd/chat/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
COPY --from=builder /app/chat-service /app/chat-service
USER appuser
EXPOSE 8082
ENTRYPOINT ["/app/chat-service"]
