# Entity Relationship Diagram (ERD) — Database `pmb_identity`

Diagram ERD berikut menggambarkan skema relasi tabel pada basis data `pmb_identity` yang dikelola secara eksklusif oleh **Identity Service**.

```mermaid
erDiagram
    PERSONS ||--o{ ADDRESSES : "memiliki"
    PERSONS ||--o{ GUARDIANS : "memiliki"
    PERSONS ||--o{ EDUCATION_HISTORIES : "memiliki"
    PERSONS ||--o{ KTP_SCANS : "memiliki riwayat"

    PERSONS {
        uuid user_id PK "UUID dari Auth Service"
        varchar full_name "Nama lengkap resmi"
        text nik "NIK terenkripsi AES-GCM (base64)"
        varchar nik_hash UK "SHA-256 hexdigest NIK plain"
        varchar birth_place "Tempat lahir"
        date birth_date "Tanggal lahir"
        varchar gender "Jenis kelamin (L / P)"
        varchar religion "Agama"
        varchar nationality "Kewarganegaraan (default WNI)"
        varchar phone "Nomor telepon / WhatsApp"
        varchar photo_object_key "Path MinIO pmb-identity-photos"
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    ADDRESSES {
        uuid id PK
        uuid user_id FK "References PERSONS(user_id)"
        varchar type "ktp / domisili (Unique per user+type)"
        text street "Alamat jalan"
        varchar rt "RT"
        varchar rw "RW"
        varchar village "Kelurahan / Desa"
        varchar district "Kecamatan"
        varchar city "Kota / Kabupaten"
        varchar province "Provinsi"
        varchar postal_code "Kode pos"
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    GUARDIANS {
        uuid id PK
        uuid user_id FK "References PERSONS(user_id)"
        varchar relation "ayah / ibu / wali (Unique per user+relation)"
        varchar name "Nama orang tua / wali"
        varchar occupation "Pekerjaan"
        varchar income_range "Rentang penghasilan"
        varchar phone "Nomor telepon"
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    EDUCATION_HISTORIES {
        uuid id PK
        uuid user_id FK "References PERSONS(user_id)"
        varchar level "Jenjang (SMA, SMK, MA, D3, dll)"
        varchar school_name "Nama sekolah / institusi"
        varchar npsn "NPSN sekolah"
        varchar major "Jurusan / peminatan"
        integer graduation_year "Tahun kelulusan"
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    KTP_SCANS {
        uuid id PK
        uuid user_id "UUID pemilik"
        varchar object_key "Path MinIO pmb-identity-ktp"
        varchar job_id "ID antrean job OCR"
        varchar status "processing, completed, retake_required, failed"
        numeric quality_score "Skor kualitas citra 0-100"
        boolean enhanced "Apakah citra ditingkatkan"
        varchar reason "Alasan retake (BLURRY, TOO_DARK, dll)"
        text reason_message "Petunjuk perbaikan foto untuk pengguna"
        jsonb field_confidences "Confidence score per field (tanpa NIK)"
        text ocr_data_raw "Data saran OCR tersimpan aman"
        timestamp confirmed_at "Waktu konfirmasi pengguna"
        jsonb fields_edited_by_user "Daftar field yang diedit user"
        timestamp created_at
        timestamp updated_at
    }
```
