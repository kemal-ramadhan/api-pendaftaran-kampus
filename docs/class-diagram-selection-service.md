# Class Diagram — Selection Service (F2-M9)

Dokumen ini memuat arsitektur kelas, antarmuka, model, repository, service layer, dan controller untuk **Selection Service** (`services/selection-service`).

```mermaid
classDiagram
    direction TB

    class ProgramController {
        -svc SelectionService
        +GetRecommendation(c *gin.Context)
        +SaveRecommendation(c *gin.Context)
        +SubmitRecommendation(c *gin.Context)
    }

    class ApprovalController {
        -svc SelectionService
        +ListApprovals(c *gin.Context)
        +Approve(c *gin.Context)
        +Return(c *gin.Context)
    }

    class AnnouncementController {
        -svc SelectionService
        +PublishAnnouncement(c *gin.Context)
    }

    class StudentController {
        -svc SelectionService
        +GetMyResult(c *gin.Context)
    }

    class InternalController {
        -resultRepo ResultRepository
        +GetResultByApplicationID(c *gin.Context)
    }

    class HealthController {
        +HealthCheck(c *gin.Context)
    }

    class SelectionService {
        <<interface>>
        +GetProgramRecommendation(ctx context.Context, periodID, pathID uuid.UUID, programCode string, userRoles, userScopes []string) (*Recommendation, []SummaryParticipantItem, error)
        +SaveProgramRecommendation(ctx context.Context, payload *SaveRecommendationPayload, actorUserID uuid.UUID, userRoles, userScopes []string) (*Recommendation, error)
        +SubmitProgramRecommendation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID, userRoles, userScopes []string) (*Recommendation, error)
        +ListPendingApprovals(ctx context.Context, periodID uuid.UUID) ([]Recommendation, error)
        +ApproveRecommendation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID) (*Recommendation, error)
        +ReturnRecommendation(ctx context.Context, id uuid.UUID, note string, actorUserID uuid.UUID) (*Recommendation, error)
        +PublishAnnouncement(ctx context.Context, payload *PublishAnnouncementPayload, actorUserID uuid.UUID) (*Announcement, error)
        +GetMyResult(ctx context.Context, userID uuid.UUID) (*MyResultResponse, error)
    }

    class selectionServiceImpl {
        -cfg *Config
        -recRepo RecommendationRepository
        -annRepo AnnouncementRepository
        -resultRepo ResultRepository
        -scoringCli internalclient.Client
        -admissionCli internalclient.Client
        -financeCli internalclient.Client
        -notificationCli internalclient.Client
        -identityCli internalclient.Client
    }

    SelectionService <|.. selectionServiceImpl : implements
    ProgramController --> SelectionService : uses
    ApprovalController --> SelectionService : uses
    AnnouncementController --> SelectionService : uses
    StudentController --> SelectionService : uses
    InternalController --> ResultRepository : uses

    class RecommendationRepository {
        <<interface>>
        +FindByPeriodPathAndProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (*Recommendation, error)
        +FindByID(ctx context.Context, id uuid.UUID) (*Recommendation, error)
        +Save(ctx context.Context, rec *Recommendation) error
        +UpdateStatus(ctx context.Context, id uuid.UUID, status RecommendationStatus, actorID *uuid.UUID, note string) error
        +ListPendingApprovals(ctx context.Context, periodID uuid.UUID) ([]Recommendation, error)
        +CountPendingApprovalsByPeriod(ctx context.Context, periodID uuid.UUID) (int64, error)
        +FindApprovedByPeriod(ctx context.Context, periodID uuid.UUID) ([]Recommendation, error)
    }

    class AnnouncementRepository {
        <<interface>>
        +Create(ctx context.Context, ann *Announcement) error
        +FindByPeriodID(ctx context.Context, periodID uuid.UUID) (*Announcement, error)
        +FindLatestPublished(ctx context.Context) (*Announcement, error)
    }

    class ResultRepository {
        <<interface>>
        +BatchCreate(ctx context.Context, results []Result) error
        +FindByApplicationID(ctx context.Context, appID uuid.UUID) (*Result, error)
        +FindByUserID(ctx context.Context, userID uuid.UUID) (*Result, error)
    }

    selectionServiceImpl --> RecommendationRepository : uses
    selectionServiceImpl --> AnnouncementRepository : uses
    selectionServiceImpl --> ResultRepository : uses

    class Recommendation {
        +uuid.UUID ID
        +uuid.UUID PeriodID
        +uuid.UUID PathID
        +string StudyProgramCode
        +RecommendationStatus Status
        +*uuid.UUID SubmittedBy
        +*time.Time SubmittedAt
        +*uuid.UUID ApprovedBy
        +*time.Time ApprovedAt
        +string Note
        +[]RecommendationItem Items
        +time.Time CreatedAt
        +time.Time UpdatedAt
    }

    class RecommendationItem {
        +uuid.UUID ID
        +uuid.UUID RecommendationID
        +uuid.UUID ApplicationID
        +DecisionType Decision
        +float64 FinalScore
        +int Rank
        +time.Time CreatedAt
        +time.Time UpdatedAt
    }

    class Announcement {
        +uuid.UUID ID
        +uuid.UUID PeriodID
        +string Title
        +string Content
        +time.Time PublishedAt
        +uuid.UUID PublishedBy
        +time.Time CreatedAt
        +time.Time UpdatedAt
    }

    class Result {
        +uuid.UUID ID
        +uuid.UUID ApplicationID
        +uuid.UUID UserID
        +DecisionType Decision
        +string AcceptedProgram
        +time.Time AnnouncedAt
        +time.Time CreatedAt
        +time.Time UpdatedAt
    }

    Recommendation "1" *-- "*" RecommendationItem : contains
```
