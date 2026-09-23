# Class Diagram — Student Service (F2-M11 Bagian Student)

Dokumen ini memuat diagram kelas dan dependensi antarmuka untuk **Student Service** (`services/student-service`).

```mermaid
classDiagram
    direction TB

    class StudentController {
        -svc StudentService
        +GetMyReregistration(c *gin.Context)
        +UpdateMyReregistration(c *gin.Context)
        +CompleteReregistration(c *gin.Context)
        +ListStudents(c *gin.Context)
        +ExportStudentsXLSX(c *gin.Context)
        +InternalGetStudentByApplicationID(c *gin.Context)
        +InternalUpdateNIM(c *gin.Context)
    }

    class StudentService {
        <<interface>>
        +GetMyReregistration(ctx context.Context, userID uuid.UUID) (*ReregistrationDetailResponse, error)
        +UpdateMyReregistration(ctx context.Context, userID uuid.UUID, req *UpdateReregistrationRequest) (*ReregistrationDetailResponse, error)
        +CompleteReregistration(ctx context.Context, userID uuid.UUID) (*Student, error)
        +ListStudents(ctx context.Context, filter StudentFilter, userRoles, userScopes []string) ([]Student, int64, error)
        +ExportStudentsXLSX(ctx context.Context, filter StudentFilter, userRoles, userScopes []string) ([]byte, error)
        +GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*Student, error)
        +GetStudentByApplicationID(ctx context.Context, appID uuid.UUID) (*Student, error)
        +UpdateStudentNIM(ctx context.Context, appID uuid.UUID, nim string) error
    }

    class StudentRepository {
        <<interface>>
        +DB() *gorm.DB
        +GetReregistrationByUserID(ctx context.Context, userID uuid.UUID) (*Reregistration, error)
        +GetReregistrationByApplicationID(ctx context.Context, appID uuid.UUID) (*Reregistration, error)
        +CreateReregistration(ctx context.Context, r *Reregistration) error
        +UpdateReregistration(ctx context.Context, r *Reregistration) error
        +UpdateReregistrationInTx(tx *gorm.DB, r *Reregistration) error
        +CreateStudentInTx(tx *gorm.DB, s *Student) error
        +GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*Student, error)
        +GetStudentByApplicationID(ctx context.Context, appID uuid.UUID) (*Student, error)
        +ListStudents(ctx context.Context, filter StudentFilter) ([]Student, int64, error)
        +UpdateStudentNIMInTx(tx *gorm.DB, appID uuid.UUID, nim string) error
        +CreateOutboxEventInTx(tx *gorm.DB, event *OutboxEvent) error
    }

    class StudentExporter {
        <<interface>>
        +ExportStudents(students []Student) ([]byte, error)
    }

    class InternalClient {
        <<interface>>
        +Get(ctx context.Context, path string, query map[string]string, dest interface{}) error
        +Post(ctx context.Context, path string, body interface{}, dest interface{}) error
        +Put(ctx context.Context, path string, body interface{}, dest interface{}) error
    }

    class Reregistration {
        +uuid.UUID ID
        +uuid.UUID ApplicationID
        +uuid.UUID UserID
        +string AcceptedProgram
        +ReregistrationStatus Status
        +string JacketSize
        +string EmergencyContactName
        +string EmergencyContactPhone
        +bool StatementAgreed
        +time.Time CompletedAt
    }

    class Student {
        +uuid.UUID ID
        +uuid.UUID UserID
        +uuid.UUID ApplicationID
        +string NIM
        +string StudyProgramCode
        +int EntryYear
        +string EntryPath
        +StudentStatus Status
    }

    class OutboxEvent {
        +uuid.UUID ID
        +string EventType
        +string Payload
        +string Status
        +int RetryCount
    }

    StudentController --> StudentService
    StudentService ..> StudentRepository
    StudentService ..> StudentExporter
    StudentService ..> InternalClient
    StudentService ..> Reregistration
    StudentService ..> Student
    StudentService ..> OutboxEvent
```
