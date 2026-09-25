# Entity Relationship Diagram (ERD) — Organization Service

Dokumen ini memuat ERD (*Entity Relationship Diagram*) untuk skema basis data PostgreSQL **`pmb_organization`** pada **Organization Service**.

---

## 1. Diagram ERD (Mermaid)

```mermaid
erDiagram
    faculties ||--o{ study_programs : "memiliki (1:N)"

    faculties {
        serial id PK "Identifier unik fakultas"
        varchar_20 code UK "Kode fakultas unik (misal: FT, FEB)"
        varchar_100 name "Nama lengkap fakultas"
        boolean is_active "Status operasional fakultas"
        timestamp_tz created_at "Waktu pembuatan data"
        timestamp_tz updated_at "Waktu pembaruan data"
        timestamp_tz deleted_at "Soft delete marker"
    }

    study_programs {
        serial id PK "Identifier unik program studi"
        integer faculty_id FK "Referensi foreign key ke faculties(id)"
        varchar_20 code UK "Kode unik prodi / scope_id (misal: TI, SI)"
        varchar_5 nim_code "Kode 2 digit penomoran NIM"
        varchar_100 name "Nama program studi"
        varchar_10 degree "Jenjang pendidikan (D3, S1, S2, S3)"
        varchar_30 accreditation "Peringkat akreditasi (Unggul, A, dsb)"
        boolean is_active "Status penerimaan pendaftaran"
        timestamp_tz created_at "Waktu pembuatan data"
        timestamp_tz updated_at "Waktu pembaruan data"
        timestamp_tz deleted_at "Soft delete marker"
    }
```

---

## 2. Indeks & Batasan Basis Data

| Tabel | Nama Indeks | Kolom | Tipe | Keterangan |
|---|---|---|---|---|
| `faculties` | `faculties_pkey` | `id` | PRIMARY KEY | Identitas unik |
| `faculties` | `faculties_code_key` | `code` | UNIQUE | Mencegah duplikasi kode fakultas |
| `faculties` | `idx_faculties_deleted_at` | `deleted_at` | B-TREE | Optimasi soft delete query |
| `study_programs` | `study_programs_pkey` | `id` | PRIMARY KEY | Identitas unik |
| `study_programs` | `study_programs_code_key` | `code` | UNIQUE | Digunakan sebagai `scope_id` di Auth |
| `study_programs` | `idx_study_programs_faculty_id` | `faculty_id` | B-TREE | Optimasi relasi join fakultas |
| `study_programs` | `idx_study_programs_code` | `code` | B-TREE | Optimasi query pencarian kode prodi |
| `study_programs` | `idx_study_programs_deleted_at` | `deleted_at` | B-TREE | Optimasi soft delete query |
