package identity

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/crypto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/service"
	"gorm.io/gorm"
)

type Module struct {
	IdentityService service.IdentityService
}

func Init(r *gin.Engine, baseDB *gorm.DB, storageCli storage.Client, ocrClient internalclient.Client, gatewaySecret, internalToken string) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "identity")
	_ = db.AutoMigrate(
		&model.Person{},
		&model.Address{},
		&model.Guardian{},
		&model.EducationHistory{},
		&model.KTPScan{},
	)
	_ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_addresses_user_type ON addresses(user_id, type)`).Error
	_ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_guardians_user_relation ON guardians(user_id, relation)`).Error

	encryptionKey := crypto.DeriveKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	personRepo := repository.NewPersonRepository(db)
	addressRepo := repository.NewAddressRepository(db)
	guardianRepo := repository.NewGuardianRepository(db)
	educationRepo := repository.NewEducationRepository(db)
	ktpScanRepo := repository.NewKTPScanRepository(db)

	identitySvc := service.NewIdentityService(personRepo, addressRepo, guardianRepo, educationRepo, storageCli, encryptionKey, "pmb-identity-photos")
	completenessSvc := service.NewCompletenessService(personRepo)
	ktpScanSvc := service.NewKTPScanService(ktpScanRepo, identitySvc, storageCli, ocrClient, "pmb-identity-ktp")

	healthCtrl := controller.NewHealthController(db)
	userCtrl := controller.NewUserController(identitySvc, completenessSvc, ktpScanSvc)
	adminCtrl := controller.NewAdminController(identitySvc, completenessSvc)
	internalCtrl := controller.NewInternalController(identitySvc, completenessSvc)

	route.RegisterRoutes(r, route.RouterConfig{
		GatewaySecret: gatewaySecret,
		InternalToken: internalToken,
	}, healthCtrl, userCtrl, adminCtrl, internalCtrl)

	return &Module{IdentityService: identitySvc}, nil
}
