# Class Diagram Auth Service

Dokumen ini memuat representasi diagram kelas (*class diagram*) untuk layanan **Auth Service** berdasarkan kode aktual di `services/auth-service` dan modul bersama di `pkg/`. Karena bahasa Go tidak memiliki konsep *class*, diagram ini menerapkan konvensi pemetaan UML:
- **`struct`** direpresentasikan sebagai kelas (*class*).
- **`interface`** ditandai dengan stereotipe `<<interface>>`.
- **Injeksi dependensi / field struct** digambarkan sebagai asosiasi berarah (`-->`).
- **Implementasi antarmuka** digambarkan sebagai realisasi (`..|>`).

Dokumen dipecah menjadi 3 diagram modular agar mudah ditelaah dan dipahami:
1. **Domain Model** (Entitas GORM dan relasi basis data)
2. **Arsitektur Layer** (Controller → Service → Repository & Layanan Infrastruktur)
3. **Middleware & Shared Package** (Middleware keamanan, rate limiter, dan helper terpusat)

---

## 1. Domain Model

Diagram berikut menggambarkan entitas domain basis data PostgreSQL yang didefinisikan pada paket `internal/model`. Entitas utama `User` terhubung secara relasional dengan peran (`UserRole`), kode OTP verifikasi (`UserOTP`), token reset kata sandi (`PasswordReset`), dan riwayat jejak audit (`AuditLog`). Relasi ini mencerminkan struktur foreign key dan relasi ORM GORM aktual pada skema basis data PMB.

![Domain Model Diagram](diagrams/auth-domain-model.svg)

```mermaid
classDiagram
    direction TB

    class User {
        +UUID ID
        +string Email
        +string Phone
        +string PasswordHash
        +string FullName
        +string Status
        +Time EmailVerifiedAt
        +Time LastLoginAt
        +Time PasswordChangedAt
        +Time CreatedAt
        +Time UpdatedAt
        +DeletedAt DeletedAt
        +BeforeCreate(tx) error
    }

    class Role {
        +uint ID
        +string Code
        +string Name
        +string Description
        +Time CreatedAt
    }

    class UserRole {
        +UUID UserID
        +uint RoleID
        +string ScopeType
        +string ScopeID
        +Time CreatedAt
        +UUID CreatedBy
    }

    class UserOTP {
        +UUID ID
        +UUID UserID
        +string Purpose
        +string Channel
        +string CodeHash
        +Time ExpiresAt
        +int16 Attempts
        +int16 MaxAttempts
        +Time ConsumedAt
        +Time CreatedAt
        +BeforeCreate(tx) error
    }

    class PasswordReset {
        +UUID ID
        +UUID UserID
        +string TokenHash
        +Time ExpiresAt
        +Time UsedAt
        +Time CreatedAt
        +BeforeCreate(tx) error
    }

    class AuditLog {
        +uint64 ID
        +UUID UserID
        +UUID ActorID
        +string Action
        +string IP
        +string UserAgent
        +string Metadata
        +Time CreatedAt
    }

    User "1" -- "0..*" UserRole : memiliki peran (UserRoles)
    Role "1" -- "0..*" UserRole : dikaitkan (Role)
    User "1" -- "0..*" UserOTP : menerima (User)
    User "1" -- "0..*" PasswordReset : meminta (User)
    User "0..1" -- "0..*" AuditLog : target aktivitas (UserID)
    User "0..1" -- "0..*" AuditLog : pelaku aktivitas (ActorID)
```

---

## 2. Arsitektur Layer

Diagram ini memetakan arsitektur berlapis *clean architecture* pada Auth Service yang terdiri atas lapisan Controller (HTTP transport Gin), Lapisan Business Logic (Service), dan Lapisan Data Access (Repository). Setiap service dan repository diikat oleh antarmuka (*interface*) untuk memfasilitasi *dependency injection*, *inversion of control*, dan kemudahan pengujian unit (*mocking*). Selain itu, diagram mencakup komponen utilitas penting seperti penerbitan token JWT RS256 (`TokenManager`), penyimpanan sesi Redis (`SessionStore`), serta pengiriman surel (`Mailer`).

![Architecture Layers Diagram](diagrams/auth-architecture-layers.svg)

```mermaid
classDiagram
    direction TB

    %% Presentation Layer / Controllers
    class AuthController {
        -AuthService authService
        -TokenManager tokenManager
        +Register(c)
        +VerifyOTP(c)
        +ResendOTP(c)
        +Login(c)
        +Refresh(c)
        +GetMe(c)
        +UpdateProfile(c)
        +ChangePassword(c)
        +ForgotPassword(c)
        +ResetPassword(c)
        +Logout(c)
        +LogoutAll(c)
        +GetJWKS(c)
    }

    class AdminController {
        -AdminService adminService
        +GetRoles(c)
        +ListUsers(c)
        +CreateUser(c)
        +GetUser(c)
        +UpdateUser(c)
        +UpdateUserRoles(c)
        +UpdateUserStatus(c)
    }

    class InternalController {
        -AdminService adminService
        +GetUser(c)
        +GetBatchUsers(c)
        +AddUserRole(c)
    }

    class HealthController {
        -DB db
        -Client rdb
        -string env
        +LivenessCheck(c)
        +HealthCheck(c)
    }

    %% Business Logic Layer / Services
    class AuthService {
        <<interface>>
        +Register(ctx, req) RegisterResponse
        +VerifyOTP(ctx, req, ip, ua) AuthResponseData
        +ResendOTP(ctx, req) error
        +Login(ctx, req, ip, ua) AuthResponseData
        +Refresh(ctx, req) AuthResponseData
        +Logout(ctx, userID, sessionID, jti, exp) error
        +LogoutAll(ctx, userID, jti, exp) error
        +GetMe(ctx, userID) UserResponseData
        +UpdateProfile(ctx, userID, req) UserResponseData
        +ChangePassword(ctx, userID, req) error
        +ForgotPassword(ctx, req) error
        +ResetPassword(ctx, req) error
    }

    class authService {
        -Config cfg
        -UserRepository userRepo
        -RoleRepository roleRepo
        -OTPRepository otpRepo
        -PasswordResetRepository resetRepo
        -AuditLogRepository auditRepo
        -TokenManager tokenManager
        -SessionStore sessionStore
        -RateLimiter rateLimiter
        -Mailer mailer
        -Logger log
    }

    class AdminService {
        <<interface>>
        +GetRoles(ctx) RoleResponseData[]
        +ListUsers(ctx, query) UserResponseData[], int64
        +CreateUser(ctx, req, actorID) UserResponseData
        +GetUser(ctx, id) UserResponseData
        +UpdateUser(ctx, id, req, actorID) UserResponseData
        +UpdateUserRoles(ctx, id, req, actorID) UserResponseData
        +UpdateUserStatus(ctx, id, req, actorID) UserResponseData
        +InternalGetUser(ctx, id) UserResponseData
        +InternalGetUsersBatch(ctx, ids) UserResponseData[]
        +InternalAddRole(ctx, id, req) UserResponseData
    }

    class adminService {
        -Config cfg
        -UserRepository userRepo
        -RoleRepository roleRepo
        -PasswordResetRepository resetRepo
        -AuditLogRepository auditRepo
        -SessionStore sessionStore
        -Mailer mailer
        -Logger logger
    }

    %% Security & Utility Services
    class TokenManager {
        <<interface>>
        +GenerateAccessToken(userID, email, name, roles, scopes, sessionID) tokenStr, jti, exp, error
        +GenerateRefreshToken(userID, sessionID) tokenStr, jti, exp, error
        +ValidateAccessToken(tokenStr) AccessClaims, error
        +ValidateRefreshToken(tokenStr) RefreshClaims, error
        +GetJWKS() JWKSResponse
        +PublicKey() PublicKey
    }

    class tokenManager {
        -Config cfg
        -PrivateKey privateKey
        -PublicKey publicKey
        -Logger log
    }

    class SessionStore {
        <<interface>>
        +SaveSession(ctx, userID, sessionID, jti, ttl) error
        +ValidateSession(ctx, userID, sessionID, jti) valid, reuseDetected, error
        +RotateSession(ctx, userID, sessionID, newJTI, ttl) error
        +RevokeSession(ctx, userID, sessionID) error
        +RevokeAllUserSessions(ctx, userID) error
        +AddToDenylist(ctx, jti, ttl) error
        +IsDenylisted(ctx, jti) bool, error
    }

    class sessionStore {
        -Client rdb
    }

    class Mailer {
        <<interface>>
        +SendOTP(ctx, toEmail, toName, otpCode) error
        +SendPasswordReset(ctx, toEmail, toName, resetLink) error
    }

    class logMailer {
        -string appName
        -Logger log
    }

    class smtpMailer {
        -Config cfg
        -Logger log
    }

    %% Data Access Layer / Repositories
    class UserRepository {
        <<interface>>
        +Create(ctx, user) error
        +FindByID(ctx, id) User, error
        +FindByEmail(ctx, email) User, error
        +Update(ctx, user) error
        +UpdatePassword(ctx, userID, passwordHash) error
        +UpdateLastLogin(ctx, userID) error
        +UpdateStatus(ctx, userID, status) error
        +ExistsByEmail(ctx, email) bool, error
        +FindUsers(ctx, role, status, q, page, perPage) User[], int64, error
        +FindBatch(ctx, ids) User[], error
    }

    class userRepository {
        -DB db
    }

    class RoleRepository {
        <<interface>>
        +FindByCode(ctx, code) Role, error
        +FindAll(ctx) Role[], error
        +AssignRole(ctx, userRole) error
        +GetUserRoles(ctx, userID) UserRole[], error
        +RemoveUserRoles(ctx, userID) error
        +ReplaceUserRoles(ctx, userID, roles) error
        +CountActiveSuperadmins(ctx) int64, error
    }

    class roleRepository {
        -DB db
    }

    class OTPRepository {
        <<interface>>
        +Create(ctx, otp) error
        +FindLatestActive(ctx, userID, purpose) UserOTP, error
        +IncrementAttempts(ctx, otpID) int16, error
        +MarkConsumed(ctx, otpID) error
        +InvalidateActiveOTPs(ctx, userID, purpose) error
    }

    class otpRepository {
        -DB db
    }

    class PasswordResetRepository {
        <<interface>>
        +Create(ctx, reset) error
        +FindByTokenHash(ctx, tokenHash) PasswordReset, error
        +MarkUsed(ctx, resetID) error
        +InvalidateActiveTokens(ctx, userID) error
    }

    class passwordResetRepository {
        -DB db
    }

    class AuditLogRepository {
        <<interface>>
        +Create(ctx, log) error
    }

    class auditLogRepository {
        -DB db
    }

    %% Relationships - Controllers to Services
    AuthController --> AuthService : uses
    AuthController --> TokenManager : uses
    AdminController --> AdminService : uses
    InternalController --> AdminService : uses

    %% Realizations - Services
    authService ..|> AuthService : implements
    adminService ..|> AdminService : implements

    %% Dependencies - Services to Repositories & Utilities
    authService --> UserRepository : userRepo
    authService --> RoleRepository : roleRepo
    authService --> OTPRepository : otpRepo
    authService --> PasswordResetRepository : resetRepo
    authService --> AuditLogRepository : auditRepo
    authService --> TokenManager : tokenManager
    authService --> SessionStore : sessionStore
    authService --> Mailer : mailer

    adminService --> UserRepository : userRepo
    adminService --> RoleRepository : roleRepo
    adminService --> PasswordResetRepository : resetRepo
    adminService --> AuditLogRepository : auditRepo
    adminService --> SessionStore : sessionStore
    adminService --> Mailer : mailer

    %% Realizations - Repositories & Utilities
    userRepository ..|> UserRepository : implements
    roleRepository ..|> RoleRepository : implements
    otpRepository ..|> OTPRepository : implements
    passwordResetRepository ..|> PasswordResetRepository : implements
    auditLogRepository ..|> AuditLogRepository : implements
    tokenManager ..|> TokenManager : implements
    sessionStore ..|> SessionStore : implements
    logMailer ..|> Mailer : implements
    smtpMailer ..|> Mailer : implements
```

---

## 3. Middleware & Shared Packages

Diagram berikut memperlihatkan komponen filter HTTP (*middleware*) serta pustaka terpusat (*shared packages*) yang melindungi rute Auth Service. Komponen mencakup verifikasi HMAC gateway signature (`pkg/gatewayauth`), autentikasi bearer token JWT dan pencegahan penggunaan token tercabut (`middleware.AuthMiddleware`), proteksi RBAC (`middleware.RequireRole`), token antar-layanan (`middleware.InternalAuthMiddleware`), serta pelacak request ID (`pkg/requestid`) dan pembangun format respons/validasi terstandarisasi (`pkg/response`).

![Middleware and Shared Packages Diagram](diagrams/auth-middleware-shared.svg)

```mermaid
classDiagram
    direction TB

    %% Middleware Layer
    class GatewayAuthMiddleware {
        <<function>>
        +Middleware(sharedSecret) HandlerFunc
    }

    class AuthMiddleware {
        <<function>>
        +AuthMiddleware(tm, store) HandlerFunc
        +GetUserID(c) UUID, bool
        +GetSessionID(c) string
        +GetTokenJTI(c) string
        +GetTokenExp(c) Time
    }

    class RoleMiddleware {
        <<function>>
        +RequireRole(roles...) HandlerFunc
        +RequireAnyRole(roles...) HandlerFunc
    }

    class InternalAuthMiddleware {
        <<function>>
        +InternalAuthMiddleware(secretToken) HandlerFunc
    }

    class RequestIDMiddleware {
        <<function>>
        +Middleware() HandlerFunc
        +Get(c) string
    }

    %% Rate Limiting Component
    class RateLimiter {
        <<interface>>
        +IsLoginLocked(ctx, email) bool, Duration, error
        +RecordLoginFailure(ctx, email) bool, int, error
        +ResetLoginFailure(ctx, email) error
        +CheckResendOTPCooldown(ctx, email) bool, Duration, error
        +SetResendOTPCooldown(ctx, email) error
        +CheckForgotPasswordCooldown(ctx, email) bool, Duration, error
        +SetForgotPasswordCooldown(ctx, email) error
    }

    class rateLimiter {
        -Client rdb
    }

    %% Shared Gateway Authentication Package
    class GatewayAuth {
        <<package>>
        +string HeaderGatewaySignature
        +string HeaderGatewayTimestamp
        +string HeaderRequestID
        +string HeaderUserID
        +Duration DefaultMaxSkew
        +GenerateSignature(secret, reqID, userID, ts) string
        +VerifySignature(secret, reqID, userID, ts, sig, skew) bool
    }

    %% Shared Response & Error Handling Package
    class ResponseHelper {
        <<package>>
        +Success(c, status, msg, data)
        +SuccessWithMeta(c, status, msg, data, meta)
        +Error(c, status, code, msg, errors)
        +Unauthorized(c, code, msg)
        +Forbidden(c, code, msg)
        +NotFound(c, msg)
        +UnprocessableEntity(c, msg, errors)
        +TooManyRequests(c, code, msg)
        +InternalServerError(c, msg)
        +BindJSON(c, obj) bool
        +BindingError(c, err)
        +FormatBindingError(err) string, map
    }

    class SuccessResponse {
        +bool Success
        +string Message
        +any Data
        +Meta Meta
    }

    class ErrorResponse {
        +bool Success
        +string Message
        +string ErrorCode
        +any Errors
        +string RequestID
    }

    class Meta {
        +int Page
        +int PerPage
        +int64 Total
    }

    %% Dependencies & Associations
    GatewayAuthMiddleware ..> GatewayAuth : uses VerifySignature
    GatewayAuthMiddleware ..> ResponseHelper : writes 401 GATEWAY_SIGNATURE_INVALID
    AuthMiddleware ..> ResponseHelper : writes 401 TOKEN_INVALID / TOKEN_REVOKED
    RoleMiddleware ..> ResponseHelper : writes 403 FORBIDDEN
    InternalAuthMiddleware ..> ResponseHelper : writes 403 FORBIDDEN

    rateLimiter ..|> RateLimiter : implements

    ResponseHelper ..> SuccessResponse : builds
    ResponseHelper ..> ErrorResponse : builds
    SuccessResponse o-- Meta : optional metadata
```
