# Multi-stage Dockerfile for PMB Student Service
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git and certificates
RUN apk add --no-cache git ca-certificates tzdata

# Copy root go.mod and pkg
COPY pkg/ ./pkg/
COPY services/student-service/go.mod services/student-service/go.sum* ./services/student-service/

WORKDIR /app/services/student-service

# Download dependencies
RUN go mod download

# Copy source code
COPY services/student-service/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/student-service ./cmd/main.go

# Final Runtime Image
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl wget

# Create non-root user
RUN addgroup -S pmb && adduser -S -G pmb pmb

COPY --from=builder /app/bin/student-service /app/student-service
COPY --from=builder /app/services/student-service/migrations /app/migrations

USER pmb

EXPOSE 8091

CMD ["/app/student-service"]
