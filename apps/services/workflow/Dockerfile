FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.work go.work.sum ./
COPY packages/go-common/go.mod packages/go-common/go.sum ./packages/go-common/
COPY apps/services/workflow/go.mod apps/services/workflow/go.sum ./apps/services/workflow/
RUN go mod download
COPY packages/go-common/ ./packages/go-common/
COPY apps/services/workflow/ ./apps/services/workflow/
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/workflow-service ./apps/services/workflow/cmd/workflow/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata python3 nodejs
RUN adduser -D -g '' appuser
COPY --from=builder /app/workflow-service /app/workflow-service
USER appuser
EXPOSE 8086
ENTRYPOINT ["/app/workflow-service"]
