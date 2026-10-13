package referral

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	GatewaySecret string
	InternalToken string
	AppPort       string
}

type Module struct {
	Service service.ReferralService
}

// Implementasi Facade API
type apiImpl struct {
	svc service.ReferralService
}

func (a *apiImpl) CheckCode(ctx context.Context, code string) (*CheckCodeResult, error) {
	res, err := a.svc.CheckCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return &CheckCodeResult{
		Valid:           res.Valid,
		InstitutionName: res.InstitutionName,
		PICName:         res.PICName,
	}, nil
}

func (a *apiImpl) GetApplicationReferral(ctx context.Context, applicationID uuid.UUID) (string, error) {
	u, err := a.svc.GetMyReferral(ctx, uuid.Nil, applicationID)
	if err != nil || u == nil {
		return "", err
	}
	return u.ReferralCode.Code, nil
}

func (a *apiImpl) MarkInvoicePaidByPartner(ctx context.Context, invoiceID uuid.UUID, partnerUserID uuid.UUID) error {
	return nil
}

func Init(r *gin.Engine, baseDB *gorm.DB, cfg Config, log *zap.Logger) (*Module, API, error) {
	db := platformdb.GetModuleDB(baseDB, "referral")
	_ = db.AutoMigrate(
		&model.ReferralCode{},
		&model.ReferralCodePIC{},
		&model.ReferralUsage{},
		&model.ReferralTransaction{},
		&model.ReferralSettlement{},
		&model.ReferralCommissionPayout{},
		&model.ReferralSetting{},
	)

	repo := repository.NewReferralRepository(db)
	svc := service.NewReferralService(repo, baseDB, log)
	ctrl := controller.NewReferralController(svc)

	route.RegisterRoutes(r, ctrl)

	mod := &Module{
		Service: svc,
	}
	api := &apiImpl{
		svc: svc,
	}

	return mod, api, nil
}
