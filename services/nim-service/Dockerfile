# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY services/nim-service/go.mod services/nim-service/go.sum* ./services/nim-service/

# Download dependencies
WORKDIR /app/services/nim-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source code
WORKDIR /app
COPY pkg/ ./pkg/
COPY services/nim-service/ ./services/nim-service/

# Build binary
WORKDIR /app/services/nim-service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/nim-service ./cmd/main.go

# Production stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata wget curl

WORKDIR /app

COPY --from=builder /app/bin/nim-service .

EXPOSE 8092

RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

CMD ["./nim-service"]
