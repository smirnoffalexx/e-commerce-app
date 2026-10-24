package http

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/handler"
	"e-commerce-backend/internal/delivery/http/middleware"
)

type RouterDeps struct {
	Logger     *slog.Logger
	Production bool
	Health     *handler.HealthHandler
}

func NewRouter(d RouterDeps) *gin.Engine {
	if d.Production {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.ContextWithFallback = true
	r.HandleMethodNotAllowed = true

	r.Use(
		middleware.RequestID(),
		middleware.AccessLog(d.Logger, "/healthz", "/readyz"),
		middleware.Recovery(d.Logger),
	)

	r.GET("/healthz", d.Health.Liveness)
	r.GET("/readyz", d.Health.Readiness)

	return r
}
