package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/auth"
	"e-commerce-backend/internal/delivery/http/dto"
	"e-commerce-backend/internal/delivery/http/response"
)

type AuthHandler struct {
	auth AuthService
}

func NewAuthHandler(svc AuthService) *AuthHandler {
	return &AuthHandler{auth: svc}
}

func (h *AuthHandler) Register(r gin.IRouter, parser auth.TokenParser) {
	g := r.Group("/auth")
	g.POST("/register", h.SignUp)
	g.POST("/login", h.Login)
	g.POST("/refresh", h.Refresh)
	g.POST("/logout", h.Logout)

	session := g.Group("", auth.AuthenticateAllowingPasswordChange(parser))
	session.GET("/me", h.Me)
	session.POST("/change-password", h.ChangePassword)
}

// SignUp godoc
//
//	@Summary		Register a customer account
//	@Description	Creates a USER account. Admin accounts are created by a SUPERADMIN.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CredentialsRequest	true	"Credentials"
//	@Success		201		{object}	dto.UserResponse
//	@Failure		400		{object}	response.Problem	"Malformed JSON"
//	@Failure		409		{object}	response.Problem	"Email already registered"
//	@Failure		422		{object}	response.Problem	"Invalid email or weak password"
//	@Failure		500		{object}	response.Problem
//	@Router			/api/v1/auth/register [post]
func (h *AuthHandler) SignUp(c *gin.Context) {
	var req dto.CredentialsRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	u, err := h.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.NewUserResponse(&u))
}

// Login godoc
//
//	@Summary		Log in
//	@Description	Returns an access token (15 min) and a rotating refresh token (7 days). When password_change_required is true the access token only allows /auth/me and /auth/change-password.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CredentialsRequest	true	"Credentials"
//	@Success		200		{object}	dto.SessionResponse
//	@Failure		400		{object}	response.Problem	"Malformed JSON"
//	@Failure		401		{object}	response.Problem	"Invalid credentials or expired temporary password"
//	@Failure		500		{object}	response.Problem
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.CredentialsRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	s, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSessionResponse(&s, time.Now()))
}

// Refresh godoc
//
//	@Summary		Refresh session
//	@Description	Exchanges a refresh token for a new token pair. The old refresh token becomes invalid.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.RefreshRequest	true	"Refresh token"
//	@Success		200		{object}	dto.SessionResponse
//	@Failure		400		{object}	response.Problem	"Malformed JSON"
//	@Failure		401		{object}	response.Problem	"Invalid or reused refresh token"
//	@Failure		500		{object}	response.Problem
//	@Router			/api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	s, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSessionResponse(&s, time.Now()))
}

// Logout godoc
//
//	@Summary		Log out
//	@Description	Revokes the refresh token. Idempotent.
//	@Tags			auth
//	@Accept			json
//	@Param			body	body	dto.RefreshRequest	true	"Refresh token"
//	@Success		204
//	@Failure		400	{object}	response.Problem	"Malformed JSON"
//	@Failure		500	{object}	response.Problem
//	@Router			/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.RefreshRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Me godoc
//
//	@Summary	Current user
//	@Tags		auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	dto.UserResponse
//	@Failure	401	{object}	response.Problem	"Missing or invalid token"
//	@Failure	500	{object}	response.Problem
//	@Router		/api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	principal, _ := auth.PrincipalFrom(c)
	u, err := h.auth.Me(c.Request.Context(), principal.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUserResponse(&u))
}

// ChangePassword godoc
//
//	@Summary		Change own password
//	@Description	Available to every role, including accounts with a temporary password. Revokes all other sessions and returns a new token pair.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.ChangePasswordRequest	true	"Passwords"
//	@Success		200		{object}	dto.SessionResponse
//	@Failure		400		{object}	response.Problem	"Malformed JSON"
//	@Failure		401		{object}	response.Problem	"Missing or invalid token"
//	@Failure		422		{object}	response.Problem	"Wrong current password or weak new password"
//	@Failure		500		{object}	response.Problem
//	@Router			/api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	principal, _ := auth.PrincipalFrom(c)
	s, err := h.auth.ChangePassword(c.Request.Context(), principal.UserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSessionResponse(&s, time.Now()))
}
