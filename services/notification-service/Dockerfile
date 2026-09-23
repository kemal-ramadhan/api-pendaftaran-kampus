# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY services/notification-service/go.mod services/notification-service/go.sum* ./services/notification-service/

# Download dependencies
WORKDIR /app/services/notification-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source code
WORKDIR /app
COPY pkg/ ./pkg/
COPY services/notification-service/ ./services/notification-service/

# Build binary
WORKDIR /app/services/notification-service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/notification-service ./cmd/main.go

# Production stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata wget curl

WORKDIR /app

COPY --from=builder /app/bin/notification-service .

EXPOSE 8083

RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

CMD ["./notification-service"]
