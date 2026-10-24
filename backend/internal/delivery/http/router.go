package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	_ "e-commerce-backend/docs"
	"e-commerce-backend/internal/delivery/http/auth"
	"e-commerce-backend/internal/delivery/http/handler"
	"e-commerce-backend/internal/delivery/http/middleware"
	"e-commerce-backend/internal/delivery/http/ratelimit"
	"e-commerce-backend/internal/delivery/http/response"
	"e-commerce-backend/internal/domain"
)

type RouterDeps struct {
	Logger         *slog.Logger
	Production     bool
	Health         *handler.HealthHandler
	Products       *handler.ProductHandler
	Categories     *handler.CategoryHandler
	Auth           *handler.AuthHandler
	Admins         *handler.AdminHandler
	Imports        *handler.CatalogImportHandler
	Tokens         auth.TokenParser
	Limiter        ratelimit.Limiter
	TrustedProxies []string
}

var (
	authRateLimit = ratelimit.Policy{Name: "auth", Limit: 10, Window: time.Minute, FailOpen: false}
	apiRateLimit  = ratelimit.Policy{Name: "api", Limit: 300, Window: time.Minute, FailOpen: true}
)

// NewRouter godoc
//
//	@title						E-Commerce API
//	@version					1.0
//	@description				Product catalog, search and checkout API. Errors follow RFC 9457 (application/problem+json). All /api/v1 routes are rate limited per client IP (300 requests per minute, auth endpoints 10 per minute); limits are reported in RateLimit-* headers.
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Access token from /api/v1/auth/login, sent as "Bearer <token>".
func NewRouter(d RouterDeps) (*gin.Engine, error) {
	if d.Production {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.ContextWithFallback = true
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(d.TrustedProxies); err != nil {
		return nil, fmt.Errorf("trusted proxies: %w", err)
	}

	r.Use(
		middleware.RequestID(),
		middleware.AccessLog(d.Logger, "/healthz", "/readyz"),
		middleware.Recovery(d.Logger, func(c *gin.Context) {
			response.Write(c, response.Problem{
				Status: http.StatusInternalServerError,
				Code:   "internal_error",
				Detail: "unexpected error, please try again later",
			})
		}),
	)

	r.NoRoute(func(c *gin.Context) {
		response.Write(c, response.Problem{Status: http.StatusNotFound, Code: "route_not_found", Detail: "route does not exist"})
	})
	r.NoMethod(func(c *gin.Context) {
		response.Write(c, response.Problem{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed", Detail: "method is not allowed for this route"})
	})

	r.GET("/healthz", d.Health.Liveness)
	r.GET("/readyz", d.Health.Readiness)
	r.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	var apiLimit, authLimit []gin.HandlerFunc
	if d.Limiter != nil {
		apiLimit = []gin.HandlerFunc{ratelimit.Middleware(d.Limiter, apiRateLimit)}
		authLimit = []gin.HandlerFunc{ratelimit.Middleware(d.Limiter, authRateLimit)}
	}

	api := r.Group("/api/v1", apiLimit...)
	d.Auth.Register(api, d.Tokens, authLimit...)
	d.Products.Register(api, auth.Authenticate(d.Tokens), auth.Require(domain.PermProductsWrite))
	d.Categories.Register(api, auth.Authenticate(d.Tokens), auth.Require(domain.PermCategoriesWrite))
	d.Admins.Register(api, auth.Authenticate(d.Tokens), auth.Require(domain.PermAdminsManage))
	d.Imports.Register(api, auth.Authenticate(d.Tokens), auth.Require(domain.PermProductsImport))

	return r, nil
}
