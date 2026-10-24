package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/dto"
	"e-commerce-backend/internal/delivery/http/response"
	"e-commerce-backend/internal/usecase"
)

type AdminHandler struct {
	admins AdminService
}

func NewAdminHandler(admins AdminService) *AdminHandler {
	return &AdminHandler{admins: admins}
}

func (h *AdminHandler) Register(r gin.IRouter, guard ...gin.HandlerFunc) {
	g := r.Group("/admins", guard...)
	g.GET("", h.List)
	g.POST("", h.Create)
	g.POST("/:id/reset-password", h.ResetPassword)
	g.POST("/:id/deactivate", h.Deactivate)
	g.POST("/:id/activate", h.Activate)
}

// List godoc
//
//	@Summary	List admins
//	@Tags		admins
//	@Produce	json
//	@Security	BearerAuth
//	@Param		page		query		int	false	"Page number (1..10000)"	default(1)
//	@Param		page_size	query		int	false	"Items per page (1..100)"	default(20)
//	@Success	200			{object}	dto.UserListResponse
//	@Failure	400			{object}	response.Problem	"Malformed query parameters"
//	@Failure	401			{object}	response.Problem	"Missing or invalid token"
//	@Failure	403			{object}	response.Problem	"Only SUPERADMIN"
//	@Failure	422			{object}	response.Problem	"Paging out of range"
//	@Failure	500			{object}	response.Problem
//	@Router		/api/v1/admins [get]
func (h *AdminHandler) List(c *gin.Context) {
	q := response.NewQueryParser(c)
	req := usecase.PageRequest{Page: q.Int("page"), PageSize: q.Int("page_size")}
	if err := q.Err(); err != nil {
		response.Error(c, err)
		return
	}
	page, err := h.admins.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUserListResponse(&page))
}

// Create godoc
//
//	@Summary		Create admin
//	@Description	Creates an ADMIN with a server-generated temporary password valid for 24 hours. The password is returned only in this response; the admin must change it on first login.
//	@Tags			admins
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateAdminRequest	true	"Admin"
//	@Success		201		{object}	dto.AdminCredentialsResponse
//	@Failure		400		{object}	response.Problem	"Malformed JSON"
//	@Failure		401		{object}	response.Problem	"Missing or invalid token"
//	@Failure		403		{object}	response.Problem	"Only SUPERADMIN"
//	@Failure		409		{object}	response.Problem	"Email already registered"
//	@Failure		422		{object}	response.Problem	"Invalid email"
//	@Failure		500		{object}	response.Problem
//	@Router			/api/v1/admins [post]
func (h *AdminHandler) Create(c *gin.Context) {
	var req dto.CreateAdminRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	creds, err := h.admins.Create(c.Request.Context(), req.Email)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, dto.NewAdminCredentialsResponse(&creds))
}

// ResetPassword godoc
//
//	@Summary		Reset admin password
//	@Description	Issues a new temporary password and signs the admin out of all sessions.
//	@Tags			admins
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Admin ID"	format(uuid)
//	@Success		200	{object}	dto.AdminCredentialsResponse
//	@Failure		400	{object}	response.Problem	"Invalid ID"
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		403	{object}	response.Problem	"Only SUPERADMIN"
//	@Failure		404	{object}	response.Problem	"Admin not found"
//	@Failure		500	{object}	response.Problem
//	@Router			/api/v1/admins/{id}/reset-password [post]
func (h *AdminHandler) ResetPassword(c *gin.Context) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	creds, err := h.admins.ResetPassword(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, dto.NewAdminCredentialsResponse(&creds))
}

// Deactivate godoc
//
//	@Summary		Deactivate admin
//	@Description	Blocks login and revokes refresh tokens. Already issued access tokens expire within 15 minutes. Idempotent.
//	@Tags			admins
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Admin ID"	format(uuid)
//	@Success		200	{object}	dto.UserResponse
//	@Failure		400	{object}	response.Problem	"Invalid ID"
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		403	{object}	response.Problem	"Only SUPERADMIN"
//	@Failure		404	{object}	response.Problem	"Admin not found"
//	@Failure		500	{object}	response.Problem
//	@Router			/api/v1/admins/{id}/deactivate [post]
func (h *AdminHandler) Deactivate(c *gin.Context) {
	h.setActive(c, false)
}

// Activate godoc
//
//	@Summary		Activate admin
//	@Description	Re-enables a deactivated admin. Idempotent.
//	@Tags			admins
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Admin ID"	format(uuid)
//	@Success		200	{object}	dto.UserResponse
//	@Failure		400	{object}	response.Problem	"Invalid ID"
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		403	{object}	response.Problem	"Only SUPERADMIN"
//	@Failure		404	{object}	response.Problem	"Admin not found"
//	@Failure		500	{object}	response.Problem
//	@Router			/api/v1/admins/{id}/activate [post]
func (h *AdminHandler) Activate(c *gin.Context) {
	h.setActive(c, true)
}

func (h *AdminHandler) setActive(c *gin.Context, active bool) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	u, err := h.admins.SetActive(c.Request.Context(), id, active)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUserResponse(&u))
}
