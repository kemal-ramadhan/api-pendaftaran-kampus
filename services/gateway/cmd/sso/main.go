package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	pkglogger "github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	authconfig "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/config"
	authcontroller "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/controller"
	authlimiter "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/limiter"
	authmailer "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/mailer"
	authmigrations "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/migrations"
	authrepo "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/repository"
	authseeder "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/seeder"
	authservice "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/service"
	authtoken "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/token"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	authCfg := authconfig.LoadConfig()
	log, err := pkglogger.Init(authCfg.AppEnv)
	if err != nil {
		panic("Gagal inisialisasi logger SSO: " + err.Error())
	}
	defer log.Sync()

	log.Info("Memulai SSO Server Kampus mandiri (Auth Core)...")

	db, err := authconfig.ConnectDB(authCfg)
	if err != nil {
		log.Fatal("Gagal koneksi database SSO", zap.Error(err))
	}

	migCtx, migCancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = authmigrations.RunMigrations(migCtx, db, log)
	_ = authseeder.SeedRoles(migCtx, db, log)
	_ = authseeder.SeedPermissions(migCtx, db, log)
	_ = authseeder.SeedSuperadmin(migCtx, db, authCfg, log)
	_ = authseeder.SeedServiceAccounts(migCtx, db, log)
	migCancel()

	var rdb *redis.Client
	if authCfg.RedisAddr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     authCfg.RedisAddr,
			Password: authCfg.RedisPassword,
		})
	}

	tokenMgr, err := authtoken.NewTokenManager(authCfg, log)
	if err != nil {
		log.Fatal("Gagal inisialisasi token manager SSO", zap.Error(err))
	}
	sessionStore := authtoken.NewSessionStore(rdb)

	userRepo := authrepo.NewUserRepository(db)
	roleRepo := authrepo.NewRoleRepository(db)
	otpRepo := authrepo.NewOTPRepository(db)
	resetRepo := authrepo.NewPasswordResetRepository(db)
	auditRepo := authrepo.NewAuditLogRepository(db)
	serviceAccountRepo := authrepo.NewServiceAccountRepository(db)

	rateLimiter := authlimiter.NewRateLimiter(rdb)
	mailer := authmailer.NewMailer(authCfg, log)

	authSvc := authservice.NewAuthService(
		authCfg,
		userRepo,
		roleRepo,
		otpRepo,
		resetRepo,
		auditRepo,
		tokenMgr,
		sessionStore,
		rateLimiter,
		mailer,
		log,
		serviceAccountRepo,
	)

	ssoCtrl := authcontroller.NewSSOController(
		authCfg,
		authSvc,
		serviceAccountRepo,
		userRepo,
		tokenMgr,
		sessionStore,
		rdb,
		log,
	)

	r := gin.New()
	r.Use(pkglogger.GinRecovery(log))
	r.Use(requestid.Middleware())
	r.Use(pkglogger.GinLogger(log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "sso-core"})
	})

	r.GET("/.well-known/openid-configuration", ssoCtrl.GetOpenIDConfiguration)
	oauth2 := r.Group("/oauth2")
	{
		oauth2.GET("/authorize", ssoCtrl.Authorize)
		oauth2.POST("/token", ssoCtrl.Token)
		oauth2.GET("/userinfo", ssoCtrl.UserInfo)
		oauth2.POST("/revoke", ssoCtrl.Revoke)
		oauth2.GET("/logout", ssoCtrl.Logout)
		oauth2.POST("/logout", ssoCtrl.Logout)
	}

	sso := r.Group("/sso")
	{
		sso.GET("/login", ssoCtrl.ShowLoginPage)
		sso.POST("/login", ssoCtrl.ProcessLogin)
	}

	port := os.Getenv("SSO_PORT")
	if port == "" {
		port = "8082"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Info("SSO Server berjalan", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Gagal menjalankan SSO server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("SSO Server dihentikan dengan aman.")
}
