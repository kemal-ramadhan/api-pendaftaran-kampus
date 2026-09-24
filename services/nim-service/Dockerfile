# Multi-stage Dockerfile for PMB NIM Service
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git and certificates
RUN apk add --no-cache git ca-certificates tzdata

# Copy pkg and nim-service go.mod
COPY pkg/ ./pkg/
COPY services/nim-service/go.mod services/nim-service/go.sum* ./services/nim-service/

WORKDIR /app/services/nim-service

# Download dependencies
RUN go mod download

# Copy source code
COPY services/nim-service/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/nim-service ./cmd/main.go

# Final Runtime Image
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl wget

# Create non-root user
RUN addgroup -S pmb && adduser -S -G pmb pmb

COPY --from=builder /app/bin/nim-service /app/nim-service
COPY --from=builder /app/services/nim-service/migrations /app/migrations

USER pmb

EXPOSE 8092

CMD ["/app/nim-service"]
