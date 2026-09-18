package route

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/controller"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health", controller.HealthCheck)

	loginLimiter := middleware.RateLimiter(middleware.RateLimitOptions{
		Prefix:   "rl:login",
		Requests: 10,
		Window:   5 * time.Minute,
		KeyGetter: func(c *gin.Context) string {
			body, err := middleware.ReadBodyOnce(c)
			if err != nil || len(body) == 0 {
				return "ip:" + c.ClientIP()
			}

			var p struct {
				Email string `json:"email"`
			}
			_ = json.Unmarshal(body, &p)

			if p.Email != "" {
				return "email:" + p.Email
			}

			return "ip:" + c.ClientIP()
		},
	})

	refreshLimiter := middleware.RateLimiter(middleware.RateLimitOptions{
		Prefix:    "rl:refresh",
		Requests:  20,
		Window:    5 * time.Minute,
		UseIPOnly: true,
	})

	googleLimiter := middleware.RateLimiter(middleware.RateLimitOptions{
		Prefix:    "rl:googlecb",
		Requests:  10,
		Window:    10 * time.Minute,
		UseIPOnly: true,
	})

	auth := r.Group("/auth")
	{
		auth.POST("/register", controller.Register)
		auth.POST("/login", loginLimiter, controller.Login)
		auth.POST("/refresh", refreshLimiter, controller.Refresh)
		auth.GET("/google/login", controller.GoogleLogin)
		auth.GET("/google/callback", googleLimiter, controller.GoogleCallback)
		auth.POST("/customers/import", controller.CreateCustomerImportController)

		auth.POST("/forgot-password", controller.ForgotPassword)
		auth.POST("/reset-password", controller.ResetPassword)
	}

	authProtected := auth.Group("")
	authProtected.Use(middleware.AuthMiddleware())
	{
		authProtected.GET("/me", controller.Me)
		authProtected.POST("/verify", controller.VerifyEmail)
		authProtected.POST("/resend-otp", controller.SendOtp)
		authProtected.PUT("/profile", controller.UpdateProfile)
		authProtected.PUT("/profile/:id", controller.UpdateProfileById)
		authProtected.PUT("/change-role", controller.ChangeRole)
		authProtected.PUT("/photo-profile", controller.PhotoProfile)
		authProtected.POST("/logout", controller.Logout)
	}

	customerProtected := r.Group("/customer")
	{
		customerProtected.GET("/bulk", controller.GetCustomersBulk)
		customerProtected.GET("/by-user/:user_id", controller.GetCustomerByUserID)
		customerProtected.POST("/customers", controller.GetCustomersInternal)
		customerProtected.POST("/by-user-ids", controller.GetUserInternal)
	}

	otherProtected := r.Group("/other")
	{
		otherProtected.GET("/statistik", controller.GetStatistikCustomers)
		otherProtected.GET("/customers", controller.GetCustomersUsers)
		otherProtected.GET("/customers/:customer_id", controller.GetCustomerUserDetail)
	}
}
