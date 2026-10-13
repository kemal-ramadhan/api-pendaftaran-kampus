package organization

import (
	"context"

	"github.com/gin-gonic/gin"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Module struct {
	Service service.OrganizationService
}

func Init(r *gin.Engine, baseDB *gorm.DB, gatewaySecret, internalToken string, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "organization")
	_ = db.AutoMigrate(&model.Faculty{}, &model.StudyProgram{}, &model.InstitutionProfile{})
	_ = seeder.Seed(context.Background(), db, log)

	facultyRepo := repository.NewFacultyRepository(db)
	studyProgramRepo := repository.NewStudyProgramRepository(db)
	institutionProfileRepo := repository.NewInstitutionProfileRepository(db)
	orgService := service.NewOrganizationService(facultyRepo, studyProgramRepo, institutionProfileRepo)

	pubCtl := controller.NewPublicController(orgService)
	admCtl := controller.NewAdminController(orgService)
	internalCtl := controller.NewInternalController(orgService)
	healthCtl := controller.NewHealthController(db)

	route.SetupRouter(r, pubCtl, admCtl, internalCtl, healthCtl, gatewaySecret, internalToken)

	return &Module{Service: orgService}, nil
}
