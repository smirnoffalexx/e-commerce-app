package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/dto"
	"e-commerce-backend/internal/delivery/http/response"
)

type CategoryHandler struct {
	categories CategoryService
}

func NewCategoryHandler(categories CategoryService) *CategoryHandler {
	return &CategoryHandler{categories: categories}
}

func (h *CategoryHandler) Register(r gin.IRouter, write ...gin.HandlerFunc) {
	g := r.Group("/categories")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	w := g.Group("", write...)
	w.POST("", h.Create)
	w.PUT("/:id", h.Rename)
	w.DELETE("/:id", h.Delete)
}

// List godoc
//
//	@Summary		List categories
//	@Description	Returns all categories ordered by name.
//	@Tags			categories
//	@Produce		json
//	@Success		200	{object}	dto.CategoryListResponse
//	@Failure		500	{object}	response.Problem
//	@Router			/api/v1/categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.categories.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewCategoryListResponse(categories))
}

// Get godoc
//
//	@Summary	Get category
//	@Tags		categories
//	@Produce	json
//	@Param		id	path		string	true	"Category ID"	format(uuid)
//	@Success	200	{object}	dto.CategoryResponse
//	@Failure	400	{object}	response.Problem	"Invalid ID"
//	@Failure	404	{object}	response.Problem	"Category not found"
//	@Failure	500	{object}	response.Problem
//	@Router		/api/v1/categories/{id} [get]
func (h *CategoryHandler) Get(c *gin.Context) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	category, err := h.categories.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewCategoryResponse(&category))
}

// Create godoc
//
//	@Summary		Create category
//	@Description	Slug is derived from the name and must be unique.
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CategoryRequest	true	"Category"
//	@Success		201		{object}	dto.CategoryResponse
//	@Failure		400		{object}	response.Problem	"Malformed JSON or unknown field"
//	@Failure		409		{object}	response.Problem	"Category already exists"
//	@Failure		415		{object}	response.Problem	"Content-Type is not application/json"
//	@Failure		422		{object}	response.Problem	"Validation failed"
//	@Failure		500		{object}	response.Problem
//	@Security		BearerAuth
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		403	{object}	response.Problem	"Insufficient permissions"
//	@Router			/api/v1/categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req dto.CategoryRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	category, err := h.categories.Create(c.Request.Context(), req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Location", "/api/v1/categories/"+category.ID.String())
	c.JSON(http.StatusCreated, dto.NewCategoryResponse(&category))
}

// Rename godoc
//
//	@Summary	Rename category
//	@Tags		categories
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Category ID"	format(uuid)
//	@Param		body	body		dto.CategoryRequest	true	"Category"
//	@Success	200		{object}	dto.CategoryResponse
//	@Failure	400		{object}	response.Problem	"Malformed JSON, unknown field or invalid ID"
//	@Failure	404		{object}	response.Problem	"Category not found"
//	@Failure	409		{object}	response.Problem	"Category already exists"
//	@Failure	415		{object}	response.Problem	"Content-Type is not application/json"
//	@Failure	422		{object}	response.Problem	"Validation failed"
//	@Failure	500		{object}	response.Problem
//	@Security	BearerAuth
//	@Failure	401	{object}	response.Problem	"Missing or invalid token"
//	@Failure	403	{object}	response.Problem	"Insufficient permissions"
//	@Router		/api/v1/categories/{id} [put]
func (h *CategoryHandler) Rename(c *gin.Context) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req dto.CategoryRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	category, err := h.categories.Rename(c.Request.Context(), id, req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewCategoryResponse(&category))
}

// Delete godoc
//
//	@Summary		Delete category
//	@Description	Fails with 409 category_in_use while the category has active products.
//	@Tags			categories
//	@Param			id	path	string	true	"Category ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	response.Problem	"Invalid ID"
//	@Failure		404	{object}	response.Problem	"Category not found"
//	@Failure		409	{object}	response.Problem	"Category has products"
//	@Failure		500	{object}	response.Problem
//	@Security		BearerAuth
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		403	{object}	response.Problem	"Insufficient permissions"
//	@Router			/api/v1/categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.categories.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
