# Schema Database & Class Diagram
# Auth Service · Identity Service · Organization Service
# Draft v1 — 22 September 2026
# Format: Mermaid (render otomatis di GitHub dan draw.io → Extras → Edit Diagram)

---

## Daftar Isi
1. [Auth Service — Schema Database (ERD)](#1-auth-service--schema-database)
2. [Auth Service — Class Diagram](#2-auth-service--class-diagram)
3. [Identity Service — Schema Database (ERD)](#3-identity-service--schema-database)
4. [Identity Service — Class Diagram](#4-identity-service--class-diagram)
5. [Organization Service — Schema Database (ERD)](#5-organization-service--schema-database)
6. [Organization Service — Class Diagram](#6-organization-service--class-diagram)
7. [Hubungan Antar Service](#7-hubungan-antar-service)

---

## 1. Auth Service — Schema Database

**Database:** `pmb_auth`
Semua primary key UUID kecuali `roles` (smallserial). Soft delete pada `users` (`deleted_at`).

```mermaid
erDiagram
    users {
        uuid        id              PK
        varchar     email           "unique, lowercase"
        varchar     phone           "nullable, format +62"
        varchar     password_hash   "bcrypt cost 12"
        varchar     full_name
        varchar     status          "pending | active | inactive"
        timestamptz email_verified_at   "nullable"
        timestamptz last_login_at       "nullable"
        timestamptz password_changed_at "nullable"
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at          "nullable, soft delete"
    }

    roles {
        smallint    id      PK
        varchar     code    "unique: calon_mahasiswa | mahasiswa | admin_pmb | keuangan | dosen | prodi | pimpinan | superadmin"
        varchar     name
        text        description
        timestamptz created_at
    }

    user_roles {
        uuid        user_id     FK
        smallint    role_id     FK
        varchar     scope_type  "nullable: prodi"
        varchar     scope_id    "nullable: kode prodi, mis TI"
        timestamptz created_at
        uuid        created_by  "nullable"
    }

    user_otps {
        uuid        id          PK
        uuid        user_id     FK
        varchar     purpose     "email_verification"
        varchar     channel     "email"
        varchar     code_hash   "SHA-256(kode + secret)"
        timestamptz expires_at
        smallint    attempts    "default 0"
        smallint    max_attempts "default 5"
        timestamptz consumed_at "nullable"
        timestamptz created_at
    }

    password_resets {
        uuid        id          PK
        uuid        user_id     FK
        varchar     token_hash  "SHA-256"
        timestamptz expires_at  "30 menit"
        timestamptz used_at     "nullable"
        timestamptz created_at
    }

    audit_logs {
        bigint      id          PK
        uuid        user_id     "nullable"
        uuid        actor_id    "nullable"
        varchar     action      "mis: auth.login.success"
        varchar     ip
        text        user_agent
        jsonb       metadata
        timestamptz created_at
    }

    users ||--o{ user_roles : "punya"
    roles ||--o{ user_roles : "dimiliki"
    users ||--o{ user_otps : "punya"
    users ||--o{ password_resets : "punya"
    users ||--o{ audit_logs : "melakukan"
```

**Catatan:**
- `user_roles` unique constraint: `(user_id, role_id, scope_type, scope_id)`
- `code_hash` di `user_otps`: bukan plaintext, bukan hash sederhana — SHA-256 dari `kode + OTP_SECRET`
- `audit_logs` tidak memakai soft delete; data audit tidak boleh dihapus

---

## 2. Auth Service — Class Diagram

**Pola arsitektur:** Handler → Service (interface) → Repository (interface) → DB/Redis

```mermaid
classDiagram
    direction TB

    %% ── HANDLER / CONTROLLER ──
    class AuthHandler {
        +Register(c *gin.Context)
        +VerifyOTP(c *gin.Context)
        +ResendOTP(c *gin.Context)
        +Login(c *gin.Context)
        +Refresh(c *gin.Context)
        +ForgotPassword(c *gin.Context)
        +ResetPassword(c *gin.Context)
        +Me(c *gin.Context)
        +UpdateProfile(c *gin.Context)
        +ChangePassword(c *gin.Context)
        +Logout(c *gin.Context)
        +LogoutAll(c *gin.Context)
    }

    class AdminHandler {
        +ListRoles(c *gin.Context)
        +ListUsers(c *gin.Context)
        +CreateUser(c *gin.Context)
        +GetUser(c *gin.Context)
        +UpdateUser(c *gin.Context)
        +UpdateUserRoles(c *gin.Context)
        +UpdateUserStatus(c *gin.Context)
    }

    class InternalHandler {
        +GetUser(c *gin.Context)
        +BatchGetUsers(c *gin.Context)
        +AddUserRole(c *gin.Context)
    }

    class JWKSHandler {
        +ServeJWKS(c *gin.Context)
    }

    %% ── SERVICE ──
    class AuthService {
        <<interface>>
        +Register(ctx, req) (*TokenPair, error)
        +VerifyOTP(ctx, req) (*TokenPair, error)
        +ResendOTP(ctx, email) error
        +Login(ctx, req) (*TokenPair, error)
        +Refresh(ctx, refreshToken) (*TokenPair, error)
        +ForgotPassword(ctx, email) error
        +ResetPassword(ctx, req) error
        +Logout(ctx, userID, sid, jti, exp) error
        +LogoutAll(ctx, userID) error
        +GetProfile(ctx, userID) (*UserProfile, error)
        +UpdateProfile(ctx, userID, req) error
        +ChangePassword(ctx, userID, req) error
    }

    class AuthServiceImpl {
        -userRepo UserRepository
        -roleRepo RoleRepository
        -otpRepo  OTPRepository
        -pwdRepo  PasswordResetRepository
        -auditRepo AuditLogRepository
        -tokenMgr TokenManager
        -session  SessionStore
        -mailer   Mailer
    }

    class AdminService {
        <<interface>>
        +ListRoles(ctx) ([]Role, error)
        +ListUsers(ctx, filter) (*PaginatedUsers, error)
        +CreateUser(ctx, req) (*User, error)
        +UpdateUserRoles(ctx, userID, roles) error
        +UpdateUserStatus(ctx, userID, status) error
    }

    %% ── REPOSITORY ──
    class UserRepository {
        <<interface>>
        +Create(ctx, user) error
        +FindByEmail(ctx, email) (*User, error)
        +FindByID(ctx, id) (*User, error)
        +Update(ctx, user) error
        +FindWithRoles(ctx, id) (*User, error)
    }

    class RoleRepository {
        <<interface>>
        +FindAll(ctx) ([]Role, error)
        +FindByCode(ctx, code) (*Role, error)
        +SetUserRoles(ctx, userID, roles) error
    }

    class OTPRepository {
        <<interface>>
        +Create(ctx, otp) error
        +FindActive(ctx, userID, purpose) (*UserOTP, error)
        +MarkConsumed(ctx, id) error
        +IncrementAttempts(ctx, id) error
    }

    class PasswordResetRepository {
        <<interface>>
        +Create(ctx, pr) error
        +FindByTokenHash(ctx, hash) (*PasswordReset, error)
        +MarkUsed(ctx, id) error
    }

    class AuditLogRepository {
        <<interface>>
        +Create(ctx, log) error
    }

    %% ── TOKEN & SESSION ──
    class TokenManager {
        <<interface>>
        +GenerateTokenPair(ctx, user, roles, scopes) (*TokenPair, error)
        +ValidateAccessToken(tokenStr) (*Claims, error)
        +ValidateRefreshToken(tokenStr) (*RefreshClaims, error)
        +GetJWKS() *jose.JSONWebKeySet
    }

    class RSATokenManager {
        -privateKey *rsa.PrivateKey
        -publicKey  *rsa.PublicKey
        -kid        string
        -accessTTL  time.Duration
        -refreshTTL time.Duration
    }

    class SessionStore {
        <<interface>>
        +SaveRefresh(ctx, sub, sid, jti, ttl) error
        +ValidateRefresh(ctx, sub, sid, jti) (bool, error)
        +RotateRefresh(ctx, sub, sid, newJTI, ttl) error
        +DeleteSession(ctx, sub, sid) error
        +DeleteAllSessions(ctx, sub) error
        +AddToDenylist(ctx, jti, ttl) error
        +IsRevoked(ctx, jti) (bool, error)
    }

    class RedisSessionStore {
        -client *redis.Client
    }

    %% ── MAILER ──
    class Mailer {
        <<interface>>
        +SendOTP(ctx, to, name, otp string) error
        +SendPasswordReset(ctx, to, name, link string) error
    }

    class SMTPMailer {
        -host, port, user, pass string
        -fromAddr, fromName string
    }

    class LogMailer {
        -logger *zap.Logger
    }

    %% ── MIDDLEWARE ──
    class JWTMiddleware {
        +Handle() gin.HandlerFunc
        -validateToken(tokenStr) (*Claims, error)
        -checkDenylist(jti) bool
    }

    class GatewayAuthMiddleware {
        +Handle() gin.HandlerFunc
        -verifyHMAC(sig, reqID, userID, ts) bool
    }

    class RateLimiter {
        +Handle(key, limit, window) gin.HandlerFunc
    }

    %% ── RELASI ──
    AuthHandler --> AuthService : uses
    AdminHandler --> AdminService : uses
    AuthServiceImpl ..|> AuthService : implements
    AuthServiceImpl --> UserRepository : uses
    AuthServiceImpl --> RoleRepository : uses
    AuthServiceImpl --> OTPRepository : uses
    AuthServiceImpl --> PasswordResetRepository : uses
    AuthServiceImpl --> AuditLogRepository : uses
    AuthServiceImpl --> TokenManager : uses
    AuthServiceImpl --> SessionStore : uses
    AuthServiceImpl --> Mailer : uses
    RSATokenManager ..|> TokenManager : implements
    RedisSessionStore ..|> SessionStore : implements
    SMTPMailer ..|> Mailer : implements
    LogMailer ..|> Mailer : implements
    JWTMiddleware --> TokenManager : uses
    JWTMiddleware --> SessionStore : uses (denylist)
```

---

## 3. Identity Service — Schema Database

**Database:** `pmb_identity`
`user_id` adalah UUID dari Auth Service (bukan foreign key lintas DB).
NIK dienkripsi AES-GCM at-rest; `nik_hash` untuk pencarian unik.

```mermaid
erDiagram
    persons {
        uuid        user_id         PK "dari Auth Service"
        varchar     full_name
        varchar     nik_encrypted   "AES-GCM, nullable"
        varchar     nik_hash        "SHA-256, unique, nullable"
        varchar     birth_place     "nullable"
        date        birth_date      "nullable"
        char        gender          "L | P, nullable"
        varchar     religion        "nullable"
        varchar     nationality     "default WNI"
        varchar     phone           "nullable"
        varchar     photo_object_key "MinIO key, nullable"
        timestamptz created_at
        timestamptz updated_at
    }

    addresses {
        uuid        id          PK
        uuid        user_id     FK
        varchar     type        "ktp | domisili"
        text        street      "nullable"
        varchar     rt          "nullable"
        varchar     rw          "nullable"
        varchar     village     "nullable"
        varchar     district    "nullable"
        varchar     city        "nullable"
        varchar     province    "nullable"
        varchar     postal_code "nullable"
        timestamptz created_at
        timestamptz updated_at
    }

    guardians {
        uuid        id              PK
        uuid        user_id         FK
        varchar     relation        "ayah | ibu | wali"
        varchar     name
        varchar     occupation      "nullable"
        varchar     income_range    "nullable"
        varchar     phone           "nullable"
        timestamptz created_at
        timestamptz updated_at
    }

    education_histories {
        uuid        id              PK
        uuid        user_id         FK
        varchar     level           "SMA | SMK | MA | D3 | S1 | ..."
        varchar     school_name
        varchar     npsn            "nullable"
        varchar     major           "nullable"
        int         graduation_year "nullable"
        timestamptz created_at
        timestamptz updated_at
    }

    ktp_scans {
        uuid        id                      PK
        uuid        user_id                 FK
        varchar     object_key              "gambar asli di MinIO"
        varchar     status                  "processing | completed | retake_required | failed | confirmed"
        int         quality_score           "0-100"
        boolean     enhanced                "apakah gambar diperbaiki"
        jsonb       field_confidences       "per-field confidence, TANPA nilai NIK"
        timestamptz confirmed_at            "nullable"
        jsonb       fields_edited_by_user   "field yang diubah user dari hasil OCR"
        timestamptz created_at
    }

    persons ||--o{ addresses : "punya"
    persons ||--o{ guardians : "punya"
    persons ||--o{ education_histories : "punya"
    persons ||--o{ ktp_scans : "punya"
```

**Catatan:**
- `nik_encrypted`: kolom cipher; `nik_hash` untuk query unik (tidak bisa reverse ke NIK asli)
- `field_confidences` di `ktp_scans` menyimpan skor kepercayaan per field (nama, tanggal lahir, alamat) **tanpa menyimpan nilai NIK**
- Gambar KTP asli disimpan di MinIO bucket `identity-ktp` (private, akses via presigned URL)
- Tabel `ktp_scans` tidak menyimpan hasil OCR mentah secara permanen (hapus setelah confirmed atau TTL)

---

## 4. Identity Service — Class Diagram

```mermaid
classDiagram
    direction TB

    %% ── HANDLER ──
    class IdentityHandler {
        +GetProfile(c *gin.Context)
        +UpdateProfile(c *gin.Context)
        +GetAddresses(c *gin.Context)
        +UpsertAddress(c *gin.Context)
        +GetGuardians(c *gin.Context)
        +UpsertGuardian(c *gin.Context)
        +GetEducationHistory(c *gin.Context)
        +UpsertEducation(c *gin.Context)
        +UploadPhoto(c *gin.Context)
        +GetCompleteness(c *gin.Context)
    }

    class KTPScanHandler {
        +CreateScan(c *gin.Context)
        +GetScanResult(c *gin.Context)
        +ConfirmScan(c *gin.Context)
    }

    class InternalIdentityHandler {
        +GetPerson(c *gin.Context)
        +GetCompleteness(c *gin.Context)
        +BatchGetPersons(c *gin.Context)
    }

    class AdminIdentityHandler {
        +GetPerson(c *gin.Context)
    }

    %% ── SERVICE ──
    class IdentityService {
        <<interface>>
        +GetProfile(ctx, userID) (*Person, error)
        +UpdateProfile(ctx, userID, req) error
        +UpsertAddress(ctx, userID, req) error
        +UpsertGuardian(ctx, userID, req) error
        +UpsertEducation(ctx, userID, req) error
        +UploadPhoto(ctx, userID, file) (string, error)
        +GetCompleteness(ctx, userID) (*Completeness, error)
    }

    class KTPScanService {
        <<interface>>
        +CreateScan(ctx, userID, file) (*KTPScan, error)
        +GetResult(ctx, userID, scanID) (*KTPScanResult, error)
        +ConfirmScan(ctx, userID, scanID, fields) error
    }

    class IdentityServiceImpl {
        -personRepo     PersonRepository
        -addressRepo    AddressRepository
        -guardianRepo   GuardianRepository
        -educationRepo  EducationRepository
        -storage        FileStorage
        -encryptor      NIKEncryptor
    }

    class KTPScanServiceImpl {
        -scanRepo   KTPScanRepository
        -storage    FileStorage
        -ocrClient  OCRClient
        -personRepo PersonRepository
    }

    %% ── REPOSITORY ──
    class PersonRepository {
        <<interface>>
        +FindByUserID(ctx, userID) (*Person, error)
        +Upsert(ctx, person) error
    }

    class AddressRepository {
        <<interface>>
        +FindByUserID(ctx, userID) ([]Address, error)
        +Upsert(ctx, address) error
    }

    class GuardianRepository {
        <<interface>>
        +FindByUserID(ctx, userID) ([]Guardian, error)
        +Upsert(ctx, guardian) error
    }

    class EducationRepository {
        <<interface>>
        +FindByUserID(ctx, userID) ([]EducationHistory, error)
        +Upsert(ctx, edu) error
    }

    class KTPScanRepository {
        <<interface>>
        +Create(ctx, scan) error
        +FindByID(ctx, scanID) (*KTPScan, error)
        +UpdateStatus(ctx, scanID, status, result) error
        +MarkConfirmed(ctx, scanID) error
    }

    %% ── INFRASTRUKTUR ──
    class FileStorage {
        <<interface>>
        +Upload(ctx, bucket, key string, r io.Reader, size int64, mime string) error
        +PresignedURL(ctx, bucket, key string, expiry time.Duration) (string, error)
        +Delete(ctx, bucket, key string) error
    }

    class MinIOStorage {
        -client *minio.Client
        -endpoint string
    }

    class NIKEncryptor {
        <<interface>>
        +Encrypt(nik string) (cipher string, hash string, error)
        +Decrypt(cipher string) (string, error)
    }

    class AESGCMEncryptor {
        -key []byte "32 byte dari env"
    }

    class OCRClient {
        <<interface>>
        +SubmitKTP(ctx, file io.Reader) (jobID string, error)
        +GetResult(ctx, jobID string) (*OCRResult, error)
    }

    class HTTPOCRClient {
        -baseURL string
        -token   string
        -http    *http.Client
    }

    %% ── RELASI ──
    IdentityHandler --> IdentityService : uses
    KTPScanHandler --> KTPScanService : uses
    IdentityServiceImpl ..|> IdentityService : implements
    KTPScanServiceImpl ..|> KTPScanService : implements
    IdentityServiceImpl --> PersonRepository : uses
    IdentityServiceImpl --> AddressRepository : uses
    IdentityServiceImpl --> GuardianRepository : uses
    IdentityServiceImpl --> EducationRepository : uses
    IdentityServiceImpl --> FileStorage : uses
    IdentityServiceImpl --> NIKEncryptor : uses
    KTPScanServiceImpl --> KTPScanRepository : uses
    KTPScanServiceImpl --> FileStorage : uses
    KTPScanServiceImpl --> OCRClient : uses
    KTPScanServiceImpl --> PersonRepository : uses (saat confirm)
    MinIOStorage ..|> FileStorage : implements
    AESGCMEncryptor ..|> NIKEncryptor : implements
    HTTPOCRClient ..|> OCRClient : implements
```

---

## 5. Organization Service — Schema Database

**Database:** `pmb_organization`
`code` pada `study_programs` dipakai sebagai `scope_id` di role `prodi` (Auth Service) dan sebagai referensi di semua service lain.

```mermaid
erDiagram
    faculties {
        uuid        id          PK
        varchar     code        "unique, mis: FT, FE, FK"
        varchar     name
        boolean     is_active   "default true"
        timestamptz created_at
        timestamptz updated_at
    }

    study_programs {
        uuid        id              PK
        uuid        faculty_id      FK
        varchar     code            "unique, mis: TI, SI, AK — dipakai sebagai scope_id role prodi"
        varchar     nim_code        "2 digit untuk format NIM, mis: 55"
        varchar     name
        varchar     degree          "D3 | S1 | S2 | S3"
        varchar     accreditation   "nullable: A | B | C | Unggul | Baik Sekali | Baik"
        boolean     is_active       "default true"
        timestamptz created_at
        timestamptz updated_at
    }

    faculties ||--o{ study_programs : "memiliki"
```

**Catatan:**
- `study_programs.code` adalah nilai yang disimpan di `user_roles.scope_id` (Auth Service). Validasi konsistensi dilakukan oleh Auth Service via `GET /internal/study-programs/:code`
- `nim_code` 2 digit dipakai NIM Service saat menerbitkan NIM

---

## 6. Organization Service — Class Diagram

```mermaid
classDiagram
    direction TB

    %% ── HANDLER ──
    class OrganizationHandler {
        +ListFaculties(c *gin.Context)
        +ListStudyPrograms(c *gin.Context)
        +GetStudyProgram(c *gin.Context)
    }

    class AdminOrganizationHandler {
        +CreateFaculty(c *gin.Context)
        +UpdateFaculty(c *gin.Context)
        +ToggleFaculty(c *gin.Context)
        +CreateStudyProgram(c *gin.Context)
        +UpdateStudyProgram(c *gin.Context)
        +ToggleStudyProgram(c *gin.Context)
    }

    class InternalOrganizationHandler {
        +GetStudyProgram(c *gin.Context)
        +BatchGetStudyPrograms(c *gin.Context)
    }

    %% ── SERVICE ──
    class OrganizationService {
        <<interface>>
        +ListFaculties(ctx, onlyActive bool) ([]Faculty, error)
        +ListStudyPrograms(ctx, filter) ([]StudyProgram, error)
        +GetStudyProgram(ctx, code) (*StudyProgram, error)
        +CreateFaculty(ctx, req) (*Faculty, error)
        +UpdateFaculty(ctx, id, req) error
        +CreateStudyProgram(ctx, req) (*StudyProgram, error)
        +UpdateStudyProgram(ctx, id, req) error
        +ValidateStudyProgramCode(ctx, code) (bool, error)
    }

    class OrganizationServiceImpl {
        -facultyRepo    FacultyRepository
        -programRepo    StudyProgramRepository
        -cache          Cache
    }

    %% ── REPOSITORY ──
    class FacultyRepository {
        <<interface>>
        +FindAll(ctx, onlyActive bool) ([]Faculty, error)
        +FindByID(ctx, id) (*Faculty, error)
        +Create(ctx, faculty) error
        +Update(ctx, faculty) error
    }

    class StudyProgramRepository {
        <<interface>>
        +FindAll(ctx, filter) ([]StudyProgram, error)
        +FindByCode(ctx, code) (*StudyProgram, error)
        +FindByCodes(ctx, codes) ([]StudyProgram, error)
        +Create(ctx, program) error
        +Update(ctx, program) error
    }

    class Cache {
        <<interface>>
        +Get(ctx, key) (string, error)
        +Set(ctx, key, value string, ttl time.Duration) error
        +Del(ctx, key) error
    }

    class RedisCache {
        -client *redis.Client
    }

    %% ── RELASI ──
    OrganizationHandler --> OrganizationService : uses
    AdminOrganizationHandler --> OrganizationService : uses
    InternalOrganizationHandler --> OrganizationService : uses
    OrganizationServiceImpl ..|> OrganizationService : implements
    OrganizationServiceImpl --> FacultyRepository : uses
    OrganizationServiceImpl --> StudyProgramRepository : uses
    OrganizationServiceImpl --> Cache : uses "cache prodi aktif 10 menit"
    RedisCache ..|> Cache : implements
```

---

## 7. Hubungan Antar Service

Komunikasi hanya lewat HTTP internal (`/internal/*`) dengan header `X-Internal-Token`. Tidak ada foreign key lintas database.

```mermaid
graph LR
    FE["Frontend (Vue 3)"]
    GW["API Gateway\n:8000"]
    AUTH["Auth Service\n:8080\nDB: pmb_auth"]
    IDENT["Identity Service\n:8081\nDB: pmb_identity"]
    ORG["Organization Service\n:8082\nDB: pmb_organization"]
    OCR["OCR Service (Python)\n:8090\n(no permanent DB)"]
    REDIS[("Redis\nSession & Cache")]
    MINIO[("MinIO\nFile Storage")]
    PG[("PostgreSQL 15\n3 database")]

    FE -->|HTTPS| GW
    GW -->|validates JWT via JWKS| AUTH
    GW -->|routes /api/v1/auth| AUTH
    GW -->|routes /api/v1/identity| IDENT
    GW -->|routes /api/v1/organization| ORG

    AUTH -->|X-Internal-Token| ORG
    note1["Auth validasi scope_id\nprodi ke Organization"]
    AUTH -.-> note1 -.-> ORG

    IDENT -->|X-Internal-Token| OCR
    note2["Submit KTP, polling hasil"]
    IDENT -.-> note2 -.-> OCR

    AUTH --- REDIS
    GW --- REDIS
    ORG --- REDIS

    IDENT --- MINIO
    IDENT --- PG
    AUTH --- PG
    ORG --- PG
```

**Aturan yang tidak boleh dilanggar:**
- Tidak ada service yang membaca database service lain secara langsung
- `user_id` (UUID dari Auth) adalah ID global — service lain menyimpannya sebagai referensi biasa, bukan foreign key
- `study_programs.code` dari Organization adalah nilai acuan di seluruh sistem (Finance, Admission, dll.) — ubah kode prodi hanya lewat admin Organization
- OCR Service tidak menyimpan gambar atau data NIK secara permanen
