# Class Diagram — NIM Service

Diagram kelas ini menggambarkan struktur arsitektur berorientasi objek pada **NIM Service** (`services/nim-service`), mencakup lapisan Controller, Domain Service, Repositori Data, Inter-Service Client, dan Entitas Model.

```mermaid
classDiagram
    direction TB

    class NIMController {
        -svc NIMService
        +IssueNIM(c: *gin.Context)
        +GetIssuanceByApplicationID(c: *gin.Context)
        +ListIssuances(c: *gin.Context)
        +ListFormats(c: *gin.Context)
        +GetFormat(c: *gin.Context)
        +CreateFormat(c: *gin.Context)
        +UpdateFormat(c: *gin.Context)
    }

    class NIMService {
        <<interface>>
        +IssueNIM(ctx: Context, req: *IssueNIMRequest) (*IssueNIMResponse, error)
        +GetIssuanceByApplicationID(ctx: Context, appID: UUID) (*NIMIssuance, error)
        +ListIssuances(ctx: Context, filter: IssuanceFilter) ([]NIMIssuance, int64, error)
        +ListFormats(ctx: Context) ([]NIMFormat, error)
        +GetFormatByID(ctx: Context, id: UUID) (*NIMFormat, error)
        +CreateFormat(ctx: Context, req: *CreateNIMFormatRequest) (*NIMFormat, error)
        +UpdateFormat(ctx: Context, id: UUID, req: *UpdateNIMFormatRequest) (*NIMFormat, error)
    }

    class nimServiceImpl {
        -cfg *Config
        -repo NIMRepository
        -studCli internalclient.Client
        -authCli internalclient.Client
        -admsCli internalclient.Client
        -notfCli internalclient.Client
        -orgCli internalclient.Client
        +orchestrateDownstreamServices(ctx: Context, issuance: *NIMIssuance)
        +getProdiNimCode(ctx: Context, prodiCode: string) string
        +getPathDigit(selectionPath: string) string
    }

    class NIMRepository {
        <<interface>>
        +DB() *gorm.DB
        +GetIssuanceByApplicationID(ctx: Context, appID: UUID) (*NIMIssuance, error)
        +GetIssuanceByNIM(ctx: Context, nim: string) (*NIMIssuance, error)
        +CreateIssuanceInTx(ctx: Context, tx: *gorm.DB, issuance: *NIMIssuance) error
        +ListIssuances(ctx: Context, filter: IssuanceFilter) ([]NIMIssuance, int64, error)
        +GetNextSequenceWithLock(ctx: Context, tx: *gorm.DB, entryYear: int, studyProgramCode: string) (int, error)
        +GetFormatForProgram(ctx: Context, studyProgramCode: string) (*NIMFormat, error)
        +ListFormats(ctx: Context) ([]NIMFormat, error)
        +GetFormatByID(ctx: Context, id: UUID) (*NIMFormat, error)
        +CreateFormat(ctx: Context, format: *NIMFormat) error
        +UpdateFormat(ctx: Context, format: *NIMFormat) error
    }

    class NIMFormat {
        +UUID ID
        +*string StudyProgramCode
        +string Pattern
        +string Description
        +bool IsActive
        +Time CreatedAt
        +Time UpdatedAt
    }

    class NIMSequence {
        +UUID ID
        +int EntryYear
        +string StudyProgramCode
        +int LastNumber
        +Time CreatedAt
        +Time UpdatedAt
    }

    class NIMIssuance {
        +UUID ID
        +UUID ApplicationID
        +UUID UserID
        +string NIM
        +int EntryYear
        +string StudyProgramCode
        +string SelectionPath
        +int SequenceNumber
        +Time IssuedAt
        +Time CreatedAt
        +Time UpdatedAt
    }

    NIMController --> NIMService : "menggunakan"
    NIMService <|.. nimServiceImpl : "mengimplementasikan"
    nimServiceImpl --> NIMRepository : "mengakses data"
    nimServiceImpl --> NIMIssuance : "mengelola penerbitan"
    nimServiceImpl --> NIMSequence : "mengunci sequence pesimistik"
    nimServiceImpl --> NIMFormat : "membaca pola format"
    NIMRepository --> NIMIssuance : "CRUD"
    NIMRepository --> NIMSequence : "SELECT FOR UPDATE"
    NIMRepository --> NIMFormat : "CRUD"
```
