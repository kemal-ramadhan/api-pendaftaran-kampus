# Entity Relationship Diagram (ERD) — Admission Service

Dokumen ini memetakan relasi tabel-tabel pada basis data `pmb_admission`.

```mermaid
erDiagram
    PERIODS ||--o{ QUOTAS : "memiliki"
    PERIODS ||--o{ APPLICATIONS : "berisi"
    ADMISSION_PATHS ||--o{ DOCUMENT_REQUIREMENTS : "mensyaratkan"
    ADMISSION_PATHS ||--o{ QUOTAS : "mengalokasikan"
    ADMISSION_PATHS ||--o{ APPLICATIONS : "dipilih"
    APPLICATIONS ||--o{ APPLICATION_DOCUMENTS : "melampirkan"
    APPLICATIONS ||--o{ APPLICATION_STATUS_LOGS : "mencatat"

    PERIODS {
        serial id PK "ID Periode pendaftaran"
        varchar(100) name "Nama periode / gelombang"
        varchar(20) academic_year "Tahun akademik, cth: 2026/2027"
        timestamptz start_date "Waktu mulai pendaftaran"
        timestamptz end_date "Waktu selesai pendaftaran"
        boolean is_active "Status aktif periode"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    ADMISSION_PATHS {
        serial id PK "ID Jalur masuk"
        varchar(100) name "Nama jalur masuk"
        varchar(20) code UK "Kode jalur masuk (REG, PRES-AK, dll)"
        text description "Deskripsi jalur"
        numeric(12,2) registration_fee "Biaya pendaftaran"
        boolean requires_exam "Wajib tes atau tidak"
        boolean is_active "Status aktif jalur"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    DOCUMENT_REQUIREMENTS {
        uuid id PK "UUID Persyaratan"
        int admission_path_id FK "ID Jalur Masuk"
        varchar(50) document_type "Tipe dokumen: ijazah, rapor, ktp, kk"
        varchar(100) name "Nama berkas persyaratan"
        text description "Petunjuk unggah berkas"
        boolean is_required "Wajib atau opsional"
        int max_size_mb "Batas ukuran berkas (MB)"
        varchar(100) allowed_formats "Format file yang diizinkan"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    QUOTAS {
        uuid id PK "UUID Kuota"
        int period_id FK "ID Periode"
        int admission_path_id FK "ID Jalur"
        uuid study_program_id "ID Program Studi"
        int capacity "Kapasitas daya tampung"
        int filled_count "Jumlah kuota terisi"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    APPLICATIONS {
        uuid id PK "UUID Formulir Pendaftaran"
        varchar(50) registration_no UK "Nomor Registrasi PMB-YYYY-P-XXXXX"
        uuid user_id FK "ID Pengguna Pendaftar"
        int period_id FK "ID Periode"
        int admission_path_id FK "ID Jalur Masuk"
        uuid choice_1_id "Pilihan Prodi 1"
        uuid choice_2_id "Pilihan Prodi 2 (opsional)"
        varchar(30) status "Status pendaftaran: awaiting_payment, dll"
        uuid invoice_id "ID Tagihan Finance (opsional)"
        uuid final_choice_id "Prodi Diterima (opsional)"
        timestamptz created_at "Waktu pendaftaran"
        timestamptz updated_at "Waktu perubahan status"
    }

    APPLICATION_DOCUMENTS {
        uuid id PK "UUID Dokumen"
        uuid application_id FK "ID Pendaftaran"
        varchar(50) document_type "Jenis dokumen"
        varchar(255) file_name "Nama berkas asli"
        varchar(500) file_url "URL berkas di MinIO"
        bigint file_size "Ukuran berkas dalam bytes"
        varchar(20) status "Status verifikasi: pending/approved/rejected"
        text notes "Catatan verifikator"
        uuid verified_by "User ID verifikator"
        timestamptz verified_at "Waktu diverifikasi"
        timestamptz created_at "Waktu diunggah"
        timestamptz updated_at "Waktu perubahan"
    }

    APPLICATION_STATUS_LOGS {
        uuid id PK "UUID Log Status"
        uuid application_id FK "ID Pendaftaran"
        varchar(30) from_status "Status asal"
        varchar(30) to_status "Status tujuan"
        uuid changed_by "User ID pengubah status"
        text notes "Keterangan perubahan status"
        timestamptz created_at "Waktu pencatatan"
    }

    OUTBOX_EVENTS {
        uuid id PK "UUID Outbox Event"
        varchar(50) aggregate_type "Tipe agregat: application"
        varchar(100) aggregate_id "ID agregat"
        varchar(100) event_type "Tipe event: invoice.create"
        text payload "Payload JSON event"
        varchar(150) idempotency_key UK "Kunci idempoten"
        varchar(20) status "Status: pending, published, failed"
        int retry_count "Jumlah retry"
        text last_error "Pesan error terakhir"
        timestamptz created_at "Waktu pembuatan event"
        timestamptz processed_at "Waktu selesai diproses"
    }
```
