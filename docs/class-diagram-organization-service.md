# Class Diagram Organization Service

Dokumen ini memuat representasi diagram kelas (*class diagram*) untuk layanan **Organization Service** berdasarkan kode aktual di `services/organization-service` dan modul bersama di `pkg/`.

---

## 1. Domain Model

Diagram entitas basis data PostgreSQL pada paket `internal/model`. Relasi satu fakultas memiliki banyak program studi (*one-to-many*).

```mermaid
classDiagram
    direction TB

    class Faculty {
        +uint ID
        +string Code
        +string Name
        +bool IsActive
        +Time CreatedAt
        +Time UpdatedAt
        +DeletedAt DeletedAt
        +StudyProgram[] StudyPrograms
        +TableName() string
    }

    class StudyProgram {
        +uint ID
        +uint FacultyID
        +Faculty Faculty
        +string Code
        +string NIMCode
        +string Name
        +string Degree
        +string Accreditation
        +bool IsActive
        +Time CreatedAt
        +Time UpdatedAt
        +DeletedAt DeletedAt
        +TableName() string
    }

    Faculty "1" *-- "0..*" StudyProgram : memiliki
```

---

## 2. Arsitektur Layer (Controller, Service, Repository)

Diagram berikut menggambarkan alur dependensi Controller $\rightarrow$ Service $\rightarrow$ Repository.

```mermaid
classDiagram
    direction TB

    class PublicController {
        -OrganizationService orgService
        +ListFaculties(c *gin.Context)
        +ListStudyPrograms(c *gin.Context)
        +GetStudyProgramByCode(c *gin.Context)
    }

    class AdminController {
        -OrganizationService orgService
        +CreateFaculty(c *gin.Context)
        +UpdateFaculty(c *gin.Context)
        +PatchFaculty(c *gin.Context)
        +CreateStudyProgram(c *gin.Context)
        +UpdateStudyProgram(c *gin.Context)
        +PatchStudyProgram(c *gin.Context)
    }

    class InternalController {
        -OrganizationService orgService
        +GetStudyProgramByCode(c *gin.Context)
        +GetBatchStudyPrograms(c *gin.Context)
    }

    class OrganizationService {
        <<interface>>
        +ListFaculties(ctx, isActive) ([]FacultyResponse, error)
        +GetFaculty(ctx, id) (*FacultyResponse, error)
        +CreateFaculty(ctx, req) (*FacultyResponse, error)
        +UpdateFaculty(ctx, id, req) (*FacultyResponse, error)
        +PatchFaculty(ctx, id, req) (*FacultyResponse, error)
        +ListStudyPrograms(ctx, facultyCode, isActive) ([]StudyProgramResponse, error)
        +GetStudyProgramByCode(ctx, code) (*StudyProgramResponse, error)
        +GetStudyProgramByID(ctx, id) (*StudyProgramResponse, error)
        +GetStudyProgramsByCodes(ctx, codes) ([]StudyProgramResponse, error)
        +CreateStudyProgram(ctx, req) (*StudyProgramResponse, error)
        +UpdateStudyProgram(ctx, id, req) (*StudyProgramResponse, error)
        +PatchStudyProgram(ctx, id, req) (*StudyProgramResponse, error)
    }

    class organizationService {
        -FacultyRepository facultyRepo
        -StudyProgramRepository studyProgramRepo
    }

    class FacultyRepository {
        <<interface>>
        +FindAll(ctx, isActive) ([]Faculty, error)
        +FindByID(ctx, id) (*Faculty, error)
        +FindByCode(ctx, code) (*Faculty, error)
        +Create(ctx, faculty) error
        +Update(ctx, faculty) error
        +Delete(ctx, id) error
    }

    class StudyProgramRepository {
        <<interface>>
        +FindAll(ctx, facultyCode, isActive) ([]StudyProgram, error)
        +FindByID(ctx, id) (*StudyProgram, error)
        +FindByCode(ctx, code) (*StudyProgram, error)
        +FindByCodes(ctx, codes) ([]StudyProgram, error)
        +Create(ctx, sp) error
        +Update(ctx, sp) error
        +Delete(ctx, id) error
    }

    PublicController --> OrganizationService
    AdminController --> OrganizationService
    InternalController --> OrganizationService
    organizationService ..|> OrganizationService
    organizationService --> FacultyRepository
    organizationService --> StudyProgramRepository
```

---

## 3. Middleware & Keamanan Inter-Service

```mermaid
classDiagram
    direction LR

    class GatewayAuthMiddleware {
        +Middleware(secret string) gin.HandlerFunc
        +RequireRole(roles ...string) gin.HandlerFunc
        +GetUserID(c *gin.Context) UUID
        +GetUserRoles(c *gin.Context) []string
        +GetScopeID(c *gin.Context, role, type) string
    }

    class InternalAuthMiddleware {
        +Middleware(secretToken string) gin.HandlerFunc
    }

    class Router {
        +SetupRouter(r, pubCtl, admCtl, intCtl, healthCtl, secret, token)
    }

    Router --> GatewayAuthMiddleware : memproteksi rute downstream
    Router --> InternalAuthMiddleware : memproteksi rute /internal/*
```
