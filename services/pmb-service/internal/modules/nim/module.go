package nim

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/service"
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

type nimAPI struct {
	svc service.NIMService
}

func (a *nimAPI) IssueNIM(ctx context.Context, req IssueNIMRequest) (*IssueNIMResponse, error) {
	res, err := a.svc.IssueNIM(ctx, &model.IssueNIMRequest{
		ApplicationID:    req.ApplicationID,
		UserID:           req.UserID,
		StudyProgramCode: req.StudyProgramCode,
		EntryYear:        req.EntryYear,
		SelectionPath:    req.SelectionPath,
	})
	if err != nil {
		return nil, err
	}
	return &IssueNIMResponse{
		NIM:              res.NIM,
		StudyProgramCode: res.StudyProgramCode,
	}, nil
}

type Module struct {
	Service service.NIMService
	API     API
}

func Init(r *gin.Engine, baseDB *gorm.DB, cfg Config, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "nim")
	_ = db.AutoMigrate(
		&model.NIMFormat{},
		&model.NIMSequence{},
		&model.NIMIssuance{},
	)

	var count int64
	db.Model(&model.NIMFormat{}).Count(&count)
	if count == 0 {
		_ = db.Create(&model.NIMFormat{
			Pattern:     "{YEAR}{PRODI}{SEQ:4}",
			Description: "Format default NIM: Tahun (4 digit) + Kode Prodi + No Urut (4 digit)",
			IsActive:    true,
		}).Error
	}

	repo := repository.NewNIMRepository(db)

	internalBaseURL := "http://127.0.0.1:" + cfg.AppPort
	newClient := func(url string) internalclient.Client {
		return internalclient.NewClient(internalclient.Config{
			BaseURL:       url,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	studCli := newClient(internalBaseURL)
	authCli := newClient(cfg.AuthURL)
	admsCli := newClient(internalBaseURL)
	notfCli := newClient(internalBaseURL)
	orgCli := newClient(internalBaseURL)

	nimCfg := &config.Config{
		AppEnv:                 "development",
		Port:                   cfg.AppPort,
		GatewaySecret:          cfg.GatewaySecret,
		InternalToken:          cfg.InternalToken,
		StudentServiceURL:      internalBaseURL,
		AuthServiceURL:         cfg.AuthURL,
		AdmissionServiceURL:    internalBaseURL,
		NotificationServiceURL: internalBaseURL,
		OrganizationServiceURL: internalBaseURL,
	}

	svc := service.NewNIMService(nimCfg, repo, studCli, authCli, admsCli, notfCli, orgCli)
	ctrl := controller.NewNIMController(svc)

	route.RegisterRoutes(r, nimCfg, ctrl)

	return &Module{
		Service: svc,
		API:     &nimAPI{svc: svc},
	}, nil
}
