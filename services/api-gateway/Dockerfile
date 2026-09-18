# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY services/api-gateway/go.mod services/api-gateway/go.sum* ./services/api-gateway/

# Download dependencies
WORKDIR /app/services/api-gateway
RUN go mod download

# Copy source code
WORKDIR /app
COPY pkg/ ./pkg/
COPY services/api-gateway/ ./services/api-gateway/

# Build binary
WORKDIR /app/services/api-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api-gateway ./cmd/main.go

# Production stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/api-gateway .
COPY services/api-gateway/config/ ./config/

EXPOSE 8000

RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

CMD ["./api-gateway"]
