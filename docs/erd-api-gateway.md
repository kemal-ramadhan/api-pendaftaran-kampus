# Entity-Relationship Diagram (ERD) — API Gateway

Layanan **API Gateway** (`services/api-gateway`) didesain sebagai **stateless reverse proxy**, sehingga tidak memiliki tabel basis data relasional (PostgreSQL) tersendiri. Seluruh data autentikasi dan otorisasi dikelola oleh `auth-service`, sementara status sesi sementara disimpan di cache Redis.

## Penyimpanan State Sementara (Redis Cache & Rate Limiting)

```mermaid
erDiagram
    REDIS_RATE_LIMIT {
        string key "ratelimit:{ip}:{path}"
        int count "Jumlah request dalam window waktu aktif"
        int ttl "Time-to-live sliding window (detik)"
    }

    REDIS_TOKEN_BLACKLIST {
        string key "blacklist:token:{jti}"
        string status "revoked"
        int ttl "Sisa durasi kedaluwarsa JWT"
    }

    REDIS_SESSION_STORE {
        string key "session:{user_id}:{session_id}"
        string user_data "Payload sesi login aktif"
        int ttl "Durasi sesi aktif"
    }
```

## Relasi Logis dengan Downstream Services

API Gateway bertindak sebagai pintu gerbang tunggal (*Single Entry Point*) yang menghubungkan klien eksternal (Browser, Mobile App, Postman) ke service hilir melalui penandatanganan request HMAC:

```mermaid
flowchart TD
    Client([Klien Publik]) -->|HTTP Request| GW[API Gateway]
    GW -->|X-Gateway-Signature / HMAC| Auth[Auth Service]
    GW -->|X-Gateway-Signature / HMAC| Adm[Admission Service]
    GW -->|X-Gateway-Signature / HMAC| Fin[Finance Service]
    GW -->|X-Gateway-Signature / HMAC| Exam[Exam Service]
    GW -->|X-Gateway-Signature / HMAC| Other[Service Lainnya...]
    GW -.->|State Cache & Rate Limiting| Redis[(Redis)]
```
