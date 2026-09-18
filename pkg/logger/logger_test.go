package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestInitLogger(t *testing.T) {
	devLog, err := Init("development")
	if err != nil {
		t.Fatalf("failed to init development logger: %v", err)
	}
	if devLog == nil {
		t.Fatal("expected devLog not nil")
	}

	prodLog, err := Init("production")
	if err != nil {
		t.Fatalf("failed to init production logger: %v", err)
	}
	if prodLog == nil {
		t.Fatal("expected prodLog not nil")
	}
}

func TestGinLoggerAndRecovery(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	observedLogger := zap.New(core)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(GinRecovery(observedLogger))
	r.Use(GinLogger(observedLogger))

	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	r.GET("/panic", func(c *gin.Context) {
		panic("something went horribly wrong")
	})

	// Test OK route
	reqOK := httptest.NewRequest(http.MethodGet, "/ok", nil)
	wOK := httptest.NewRecorder()
	r.ServeHTTP(wOK, reqOK)

	if wOK.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", wOK.Code)
	}

	// Test Panic route
	reqPanic := httptest.NewRequest(http.MethodGet, "/panic", nil)
	wPanic := httptest.NewRecorder()
	r.ServeHTTP(wPanic, reqPanic)

	if wPanic.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", wPanic.Code)
	}

	allLogs := logs.All()
	if len(allLogs) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d", len(allLogs))
	}
}
