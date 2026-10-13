package admission

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/outbox"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/service"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	GatewaySecret string
	InternalToken string
	AppPort       string
	AuthURL       string
}

type Module struct {
	AdmissionService   service.AdmissionService
	ChangeService      service.ChangeService
	WithdrawalService  service.WithdrawalService
	OutboxWorkerCancel context.CancelFunc
}

func Init(r *gin.Engine, baseDB *gorm.DB, storageCli storage.Client, cfg Config, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "admission")
	_ = db.AutoMigrate(
		&model.Period{},
		&model.AdmissionPath{},
		&model.DocumentRequirement{},
		&model.Quota{},
		&model.Application{},
		&model.ApplicationDocument{},
		&model.ApplicationStatusLog{},
		&model.ApplicationChange{},
		&model.PathChangeRequest{},
		&model.Withdrawal{},
		&model.OfficialDocument{},
		&model.AdmissionOnboarding{},
		&outbox.Event{},
	)

	changeRepo := repository.NewChangeRepository(db)
	_ = changeRepo.InitApplicationChangesTriggersAndBackfill(db)
	_ = seeder.SeedAdmissionData(context.Background(), db)

	internalBaseURL := "http://127.0.0.1:" + cfg.AppPort
	newClient := func(url string) internalclient.Client {
		return internalclient.NewClient(internalclient.Config{
			BaseURL:       url,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	notifCli := newClient(internalBaseURL)
	financeCli := newClient(internalBaseURL)
	orgCli := newClient(internalBaseURL)
	identityCli := newClient(internalBaseURL)
	examCli := newClient(internalBaseURL)
	selectionCli := newClient(internalBaseURL)
	studentCli := newClient(internalBaseURL)
	nimCli := newClient(internalBaseURL)
	authCli := newClient(cfg.AuthURL)

	outboxRepo := outbox.NewRepository()
	outboxCtx, outboxCancel := context.WithCancel(context.Background())
	outboxDispatcher := service.OutboxDispatcher(financeCli)
	outbox.StartWorker(
		outboxCtx,
		db,
		outboxRepo,
		outboxDispatcher,
		2*time.Second,
		20,
		3,
	)

	periodRepo := repository.NewPeriodRepository(db)
	pathRepo := repository.NewAdmissionPathRepository(db)
	quotaRepo := repository.NewQuotaRepository(db)
	appRepo := repository.NewApplicationRepository(db)
	docRepo := repository.NewApplicationDocumentRepository(db)
	reqRepo := repository.NewDocumentRequirementRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	officialDocRepo := repository.NewOfficialDocumentRepository(db)

	admissCfg := config.Config{
		AppEnv:                 "development",
		GatewaySharedSecret:    cfg.GatewaySecret,
		InternalToken:          cfg.InternalToken,
		AuthServiceURL:         cfg.AuthURL,
		OrganizationServiceURL: internalBaseURL,
		IdentityServiceURL:     internalBaseURL,
		NotificationServiceURL: internalBaseURL,
		FinanceServiceURL:      internalBaseURL,
		ExamServiceURL:         internalBaseURL,
		SelectionServiceURL:    internalBaseURL,
		StudentServiceURL:      internalBaseURL,
		NIMServiceURL:          internalBaseURL,
		DocumentBucket:         "pmb-admission-documents",
		OutboxIntervalSeconds:  2,
		OutboxBatchSize:        20,
	}

	admissionSvc := service.NewAdmissionService(
		periodRepo,
		pathRepo,
		quotaRepo,
		appRepo,
		docRepo,
		outboxRepo,
		officialDocRepo,
		notifCli,
		identityCli,
		financeCli,
		orgCli,
		storageCli,
		admissCfg,
	)
	admissionSvc.SetOnboardingRepo(repository.NewOnboardingRepository(db))

	changeSvc := service.NewChangeService(
		changeRepo,
		appRepo,
		pathRepo,
		periodRepo,
		quotaRepo,
		docRepo,
		reqRepo,
		financeCli,
		examCli,
		notifCli,
		admissionSvc,
		withdrawalRepo,
	)

	withdrawalSvc := service.NewWithdrawalService(
		withdrawalRepo,
		appRepo,
		changeRepo,
		pathRepo,
		authCli,
		financeCli,
		examCli,
		selectionCli,
		studentCli,
		nimCli,
		identityCli,
		notifCli,
	)

	healthCtrl := controller.NewHealthController(db)
	publicCtrl := controller.NewPublicController(admissionSvc)
	appCtrl := controller.NewApplicationController(admissionSvc)
	adminCtrl := controller.NewAdminController(admissionSvc, periodRepo, pathRepo, quotaRepo, reqRepo)
	internalCtrl := controller.NewInternalController(admissionSvc)
	changeCtrl := controller.NewChangeController(changeSvc)
	withdrawalCtrl := controller.NewWithdrawalController(withdrawalSvc)

	route.RegisterRoutes(r, route.RouterConfig{
		GatewaySecret: cfg.GatewaySecret,
		InternalToken: cfg.InternalToken,
	}, healthCtrl, publicCtrl, appCtrl, adminCtrl, internalCtrl, changeCtrl, withdrawalCtrl)

	return &Module{
		AdmissionService:   admissionSvc,
		ChangeService:      changeSvc,
		WithdrawalService:  withdrawalSvc,
		OutboxWorkerCancel: outboxCancel,
	}, nil
}
