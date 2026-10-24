package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	_ "e-commerce-backend/docs"
	"e-commerce-backend/internal/delivery/http/auth"
	"e-commerce-backend/internal/delivery/http/handler"
	"e-commerce-backend/internal/delivery/http/middleware"
	"e-commerce-backend/internal/delivery/http/response"
	"e-commerce-backend/internal/domain"
)

type RouterDeps struct {
	Logger     *slog.Logger
	Production bool
	Health     *handler.HealthHandler
	Products   *handler.ProductHandler
	Categories *handler.CategoryHandler
	Auth       *handler.AuthHandler
	Admins     *handler.AdminHandler
	Tokens     auth.TokenParser
}

// NewRouter godoc
//
//	@title						E-Commerce API
//	@version					1.0
//	@description				Product catalog, search and checkout API. Errors follow RFC 9457 (application/problem+json).
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Access token from /api/v1/auth/login, sent as "Bearer <token>".
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

	api := r.Group("/api/v1")
	d.Auth.Register(api, d.Tokens)
	d.Products.Register(api, auth.Authenticate(d.Tokens), auth.Require(domain.PermProductsWrite))
	d.Categories.Register(api, auth.Authenticate(d.Tokens), auth.Require(domain.PermCategoriesWrite))
	d.Admins.Register(api, auth.Authenticate(d.Tokens), auth.Require(domain.PermAdminsManage))

	return r
}
