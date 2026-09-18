package logger

import (
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init membuat instance zap.Logger berdasarkan environment (development / production)
func Init(env string) (*zap.Logger, error) {
	var cfg zap.Config
	if strings.ToLower(env) == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	}

	return cfg.Build()
}

// GinLogger middleware Gin untuk pencatatan log akses HTTP terstruktur
func GinLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		reqID := requestid.Get(c)

		fields := []zapcore.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.String("request_id", reqID),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		if status >= http.StatusInternalServerError {
			log.Error("HTTP Server Error", fields...)
		} else if status >= http.StatusBadRequest {
			log.Warn("HTTP Client Error", fields...)
		} else {
			log.Info("HTTP Request", fields...)
		}
	}
}

// GinRecovery middleware Gin untuk penanganan panic yang aman
func GinRecovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Periksa apakah koneksi terputus
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				reqID := requestid.Get(c)
				log.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("request_id", reqID),
					zap.String("path", c.Request.URL.Path),
					zap.Stack("stack"),
				)

				if brokenPipe {
					c.Error(err.(error))
					c.Abort()
					return
				}

				response.InternalServerError(c, "Terjadi kesalahan internal pada server")
				c.Abort()
			}
		}()
		c.Next()
	}
}
