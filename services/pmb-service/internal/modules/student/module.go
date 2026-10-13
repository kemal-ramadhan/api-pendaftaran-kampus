package student

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/service"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	GatewaySecret string
	InternalToken string
	AppPort       string
}

type studentAPI struct {
	repo repository.StudentRepository
}

func (a *studentAPI) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*StudentDTO, error) {
	s, err := a.repo.GetStudentByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return &StudentDTO{
		ID:            s.ID,
		UserID:        s.UserID,
		ApplicationID: s.ApplicationID,
		NIM:           s.NIM,
		Status:        string(s.Status),
	}, nil
}

func (a *studentAPI) GetReregistration(ctx context.Context, appID uuid.UUID) (*ReregistrationDTO, error) {
	reg, err := a.repo.GetReregistrationByApplicationID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return nil, nil
	}
	return &ReregistrationDTO{
		ID:            reg.ID,
		ApplicationID: reg.ApplicationID,
		Status:        string(reg.Status),
	}, nil
}

type Module struct {
	Service service.StudentService
	API     API
}

func Init(r *gin.Engine, baseDB *gorm.DB, cfg Config, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "student")
	_ = db.AutoMigrate(
		&model.Reregistration{},
		&model.Student{},
		&model.PKKMBGroup{},
		&model.PKKMBAssignment{},
		&model.OutboxEvent{},
	)

	repo := repository.NewStudentRepository(db)
	exp := exporter.NewStudentExporter()

	internalBaseURL := "http://127.0.0.1:" + cfg.AppPort
	newClient := func(url string) internalclient.Client {
		return internalclient.NewClient(internalclient.Config{
			BaseURL:       url,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	admissionCli := newClient(internalBaseURL)
	selectionCli := newClient(internalBaseURL)
	financeCli := newClient(internalBaseURL)
	nimCli := newClient(internalBaseURL)
	identityCli := newClient(internalBaseURL)

	stdCfg := &config.Config{
		AppEnv:                 "development",
		Port:                   cfg.AppPort,
		GatewaySecret:          cfg.GatewaySecret,
		InternalToken:          cfg.InternalToken,
		AdmissionServiceURL:    internalBaseURL,
		SelectionServiceURL:    internalBaseURL,
		FinanceServiceURL:      internalBaseURL,
		OrganizationServiceURL: internalBaseURL,
		NIMServiceURL:          internalBaseURL,
		IdentityServiceURL:     internalBaseURL,
		NIMIssuanceMode:        "manual",
	}

	svc := service.NewStudentService(repo, exp, admissionCli, selectionCli, financeCli, nimCli, identityCli, stdCfg)
	ctrl := controller.NewStudentController(svc)

	route.RegisterRoutes(r, stdCfg, ctrl)

	return &Module{
		Service: svc,
		API:     &studentAPI{repo: repo},
	}, nil
}
