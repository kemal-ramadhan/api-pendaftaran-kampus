package scoring

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/service"
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
	ScoringService service.ScoringService
}

func Init(r *gin.Engine, baseDB *gorm.DB, cfg Config, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "scoring")
	_ = db.AutoMigrate(&model.ScoreComponent{}, &model.Score{}, &model.ScoreSummary{}, &model.ProgramLock{})
	_ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_scores_app_comp ON scores(application_id, component_code)`).Error
	_ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_program_locks_lookup ON program_locks(period_id, path_id, study_program_code)`).Error
	_ = seeder.SeedScoreComponents(context.Background(), db)

	compRepo := repository.NewComponentRepository(db)
	scoreRepo := repository.NewScoreRepository(db)
	excelExp := exporter.NewExcelExporter()

	internalBaseURL := "http://127.0.0.1:" + cfg.AppPort
	newClient := func(url string) internalclient.Client {
		return internalclient.NewClient(internalclient.Config{
			BaseURL:       url,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}
	admissionCli := newClient(internalBaseURL)
	examCli := newClient(internalBaseURL)
	identityCli := newClient(internalBaseURL)

	scoreCfg := &config.Config{
		AppEnv:                 "development",
		Port:                   cfg.AppPort,
		GatewaySharedKey:       cfg.GatewaySecret,
		InternalToken:          cfg.InternalToken,
		AdmissionServiceURL:    internalBaseURL,
		ExamServiceURL:         internalBaseURL,
		IdentityServiceURL:     internalBaseURL,
		OrganizationServiceURL: internalBaseURL,
	}

	scoringSvc := service.NewScoringService(scoreCfg, compRepo, scoreRepo, excelExp, admissionCli, examCli, identityCli)

	healthCtrl := controller.NewHealthController()
	compCtrl := controller.NewComponentController(scoringSvc)
	examinerCtrl := controller.NewExaminerController(scoringSvc)
	progCtrl := controller.NewProgramController(scoringSvc)
	intCtrl := controller.NewInternalController(scoringSvc)

	route.RegisterRoutes(r, route.RouterConfig{
		GatewaySecret: cfg.GatewaySecret,
		InternalToken: cfg.InternalToken,
		IsDevelopment: cfg.IsDevelopment,
	}, healthCtrl, compCtrl, examinerCtrl, progCtrl, intCtrl)

	return &Module{
		ScoringService: scoringSvc,
	}, nil
}
