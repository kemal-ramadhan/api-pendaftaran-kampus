# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY services/exam-service/go.mod services/exam-service/go.sum* ./services/exam-service/

# Download dependencies
WORKDIR /app/services/exam-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source code
WORKDIR /app
COPY pkg/ ./pkg/
COPY services/exam-service/ ./services/exam-service/

# Build binary
WORKDIR /app/services/exam-service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/exam-service ./cmd/main.go

# Production stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata wget curl

WORKDIR /app

COPY --from=builder /app/bin/exam-service .

EXPOSE 8087

RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

CMD ["./exam-service"]
