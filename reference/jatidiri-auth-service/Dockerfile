FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o jatidiri-auth-service ./cmd/main.go

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/jatidiri-auth-service .
COPY .env .env

EXPOSE 8080

RUN adduser -D appuser
USER appuser

CMD ["./jatidiri-auth-service"]
