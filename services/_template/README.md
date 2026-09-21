# Template Service PMB

Template ini adalah blueprint / kerangka dasar standar untuk pembuatan microservice baru pada PMB Fase 2:
- **Organization Service** (F2-M1)
- **Identity Service** (F2-M2)
- **Notification Service** (F2-M3)
- **Admission Service** (F2-M4 / F2-M6)
- **Finance Service** (F2-M5 / F2-M10)
- **Exam Service** (F2-M7)
- **Scoring Service** (F2-M8)
- **Selection Service** (F2-M9)
- **Student Service & NIM Service** (F2-M11)

## Cara Menggunakan Template
1. Salin folder `services/_template` ke `services/<nama-service>`.
2. Ubah module name di `go.mod` menjadi `github.com/pkm-asia/pmb-backend/services/<nama-service>`.
3. Daftarkan module di `go.work` root:
   ```
   go work use ./services/<nama-service>
   ```
4. Sesuaikan nama service, tabel, DTO, repository, dan business logic sesuai PRD.
5. Jalankan `go build ./services/<nama-service>/...` untuk memvalidasi.
