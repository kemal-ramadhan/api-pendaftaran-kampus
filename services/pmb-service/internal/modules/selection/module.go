package selection

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/service"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
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
	SelectionService service.SelectionService
}

func Init(r *gin.Engine, baseDB *gorm.DB, cfg Config, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "selection")
	_ = db.AutoMigrate(
		&model.Recommendation{},
		&model.RecommendationItem{},
		&model.Announcement{},
		&model.Result{},
	)
	_ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_recommendations_lookup ON recommendations(period_id, path_id, study_program_code)`).Error

	recRepo := repository.NewRecommendationRepository(db)
	annRepo := repository.NewAnnouncementRepository(db)
	resRepo := repository.NewResultRepository(db)

	internalBaseURL := "http://127.0.0.1:" + cfg.AppPort
	newClient := func(url string) internalclient.Client {
		return internalclient.NewClient(internalclient.Config{
			BaseURL:       url,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	scoringCli := newClient(internalBaseURL)
	admissionCli := newClient(internalBaseURL)
	financeCli := newClient(internalBaseURL)
	notificationCli := newClient(internalBaseURL)
	identityCli := newClient(internalBaseURL)

	selCfg := &config.Config{
		AppEnv:                 "development",
		Port:                   cfg.AppPort,
		GatewaySharedKey:       cfg.GatewaySecret,
		InternalToken:          cfg.InternalToken,
		ScoringServiceURL:      internalBaseURL,
		AdmissionServiceURL:    internalBaseURL,
		FinanceServiceURL:      internalBaseURL,
		NotificationServiceURL: internalBaseURL,
		IdentityServiceURL:     internalBaseURL,
		OrganizationServiceURL: internalBaseURL,
	}

	selectSvc := service.NewSelectionService(
		selCfg,
		recRepo,
		annRepo,
		resRepo,
		scoringCli,
		admissionCli,
		financeCli,
		notificationCli,
		identityCli,
	)

	healthCtrl := controller.NewHealthController()
	progCtrl := controller.NewProgramController(selectSvc)
	approvalCtrl := controller.NewApprovalController(selectSvc)
	annCtrl := controller.NewAnnouncementController(selectSvc)
	studCtrl := controller.NewStudentController(selectSvc)
	intCtrl := controller.NewInternalController(resRepo, selectSvc)

	route.RegisterRoutes(r, route.RouterConfig{
		GatewaySecret: cfg.GatewaySecret,
		InternalToken: cfg.InternalToken,
		IsDevelopment: cfg.IsDevelopment,
	}, healthCtrl, progCtrl, approvalCtrl, annCtrl, studCtrl, intCtrl)

	return &Module{
		SelectionService: selectSvc,
	}, nil
}
