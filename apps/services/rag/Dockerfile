FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/rag/go.mod apps/services/rag/go.sum ./apps/services/rag/
RUN go mod download
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/rag/ ./apps/services/rag/
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/rag-service ./apps/services/rag/cmd/rag/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
COPY --from=builder /app/rag-service /app/rag-service
USER appuser
EXPOSE 8083
ENTRYPOINT ["/app/rag-service"]
