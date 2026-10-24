package auth

import (
	"strings"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/response"
	"e-commerce-backend/internal/domain"
)

const ctxKeyPrincipal = "auth.principal"

var errMissingToken = domain.NewError(domain.ErrUnauthorized, "missing_token", "Authorization header with a Bearer token is required")

type TokenParser interface {
	Parse(token string) (domain.Principal, error)
}

func Authenticate(parser TokenParser) gin.HandlerFunc {
	return authenticate(parser, false)
}

func AuthenticateAllowingPasswordChange(parser TokenParser) gin.HandlerFunc {
	return authenticate(parser, true)
}

func Require(p domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := PrincipalFrom(c)
		if !ok || !principal.Role.Can(p) {
			response.Error(c, domain.ErrInsufficientPermissions)
			return
		}
		c.Next()
	}
}

func PrincipalFrom(c *gin.Context) (domain.Principal, bool) {
	v, ok := c.Get(ctxKeyPrincipal)
	if !ok {
		return domain.Principal{}, false
	}
	p, ok := v.(domain.Principal)
	return p, ok
}

func authenticate(parser TokenParser, allowPasswordChange bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			c.Header("WWW-Authenticate", `Bearer`)
			response.Error(c, errMissingToken)
			return
		}

		principal, err := parser.Parse(token)
		if err != nil {
			c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
			response.Error(c, domain.ErrInvalidToken)
			return
		}
		if principal.MustChangePassword && !allowPasswordChange {
			response.Error(c, domain.ErrPasswordChangeRequired)
			return
		}

		c.Set(ctxKeyPrincipal, principal)
		c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
