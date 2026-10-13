package exam

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	GatewaySecret string
	InternalToken string
	AppPort       string
	IsDevelopment bool
}

type Module struct {
	ExamService service.ExamService
	CBTService  service.CBTService
}

func Init(r *gin.Engine, baseDB *gorm.DB, storageCli storage.Client, cfg Config, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "exam")
	_ = db.AutoMigrate(
		&model.ExamSession{},
		&model.SessionExaminer{},
		&model.Participant{},
		&model.ExamCard{},
		&model.QuestionBank{},
		&model.Question{},
		&model.QuestionOption{},
		&model.CBTExam{},
		&model.CBTExamRule{},
		&model.CBTAttempt{},
		&model.CBTAnswer{},
		&model.CBTEvent{},
	)
	_ = seeder.SeedExamSessions(context.Background(), db)

	internalBaseURL := "http://127.0.0.1:" + cfg.AppPort
	newClient := func(url string) internalclient.Client {
		return internalclient.NewClient(internalclient.Config{
			BaseURL:       url,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}
	admissionCli := newClient(internalBaseURL)
	identityCli := newClient(internalBaseURL)
	notificationCli := newClient(internalBaseURL)
	scoringCli := newClient(internalBaseURL)

	sessionRepo := repository.NewSessionRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	cbtRepo := repository.NewCBTRepository(db)

	examCfg := &config.Config{
		AppEnv:           "development",
		Port:             cfg.AppPort,
		GatewaySharedKey: cfg.GatewaySecret,
		InternalToken:    cfg.InternalToken,
		ExamCardBucket:   "pmb-exam-cards",
	}

	examSvc := service.NewExamService(
		examCfg,
		sessionRepo,
		participantRepo,
		storageCli,
		admissionCli,
		identityCli,
		notificationCli,
	)
	cbtSvc := service.NewCBTService(cbtRepo, sessionRepo, participantRepo, scoringCli)

	healthCtrl := controller.NewHealthController()
	sessionCtrl := controller.NewSessionController(examSvc)
	participantCtrl := controller.NewParticipantController(examSvc)
	studentCtrl := controller.NewStudentController(examSvc)
	examinerCtrl := controller.NewExaminerController(examSvc)
	internalCtrl := controller.NewInternalController(examSvc)
	cbtCtrl := controller.NewCBTController(cbtSvc)

	route.RegisterRoutes(r, route.RouterConfig{
		GatewaySecret: cfg.GatewaySecret,
		InternalToken: cfg.InternalToken,
		IsDevelopment: cfg.IsDevelopment,
	}, healthCtrl, sessionCtrl, participantCtrl, studentCtrl, examinerCtrl, internalCtrl, cbtCtrl)

	return &Module{
		ExamService: examSvc,
		CBTService:  cbtSvc,
	}, nil
}
