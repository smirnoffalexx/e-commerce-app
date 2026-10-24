package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/dto"
	"e-commerce-backend/internal/delivery/http/response"
	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

type ProductHandler struct {
	products ProductService
}

func NewProductHandler(products ProductService) *ProductHandler {
	return &ProductHandler{products: products}
}

func (h *ProductHandler) Register(r gin.IRouter, write ...gin.HandlerFunc) {
	g := r.Group("/products")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	w := g.Group("", write...)
	w.POST("", h.Create)
	w.PUT("/:id", h.Update)
	w.DELETE("/:id", h.Delete)
}

// List godoc
//
//	@Summary		Search and list products
//	@Description	Without q returns products newest first. With q combines PostgreSQL full-text search over name and description (stemming, websearch syntax: "quoted phrase", -exclude, OR), trigram word similarity on the name for typos and partial words, and exact/prefix SKU match. Results are ranked by relevance unless another sort is given.
//	@Tags			products
//	@Produce		json
//	@Param			q			query		string	false	"Search text (max 200 chars)"	example(bluetooth speaker)
//	@Param			category_id	query		string	false	"Filter by category"			format(uuid)
//	@Param			min_price	query		string	false	"Minimum price, inclusive"		example(10.00)
//	@Param			max_price	query		string	false	"Maximum price, inclusive"		example(99.99)
//	@Param			in_stock	query		bool	false	"Only products with stock > 0"
//	@Param			sort		query		string	false	"Sort order; default relevance with q, newest without"	Enums(relevance,newest,price_asc,price_desc,name_asc)
//	@Param			page		query		int		false	"Page number (1..10000)"								default(1)
//	@Param			page_size	query		int		false	"Items per page (1..100)"								default(20)
//	@Success		200			{object}	dto.ProductListResponse
//	@Failure		400			{object}	response.Problem	"Malformed query parameters"
//	@Failure		422			{object}	response.Problem	"Invalid search parameters"
//	@Failure		500			{object}	response.Problem
//	@Router			/api/v1/products [get]
func (h *ProductHandler) List(c *gin.Context) {
	q := response.NewQueryParser(c)
	query := usecase.ListProductsQuery{
		Query:       c.Query("q"),
		CategoryID:  q.UUID("category_id"),
		MinPrice:    q.Decimal("min_price"),
		MaxPrice:    q.Decimal("max_price"),
		InStock:     q.Bool("in_stock"),
		Sort:        domain.ProductSort(c.Query("sort")),
		PageRequest: usecase.PageRequest{Page: q.Int("page"), PageSize: q.Int("page_size")},
	}
	if err := q.Err(); err != nil {
		response.Error(c, err)
		return
	}

	page, err := h.products.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewProductListResponse(&page))
}

// Get godoc
//
//	@Summary	Get product
//	@Tags		products
//	@Produce	json
//	@Param		id	path		string	true	"Product ID"	format(uuid)
//	@Success	200	{object}	dto.ProductResponse
//	@Failure	400	{object}	response.Problem	"Invalid ID"
//	@Failure	404	{object}	response.Problem	"Product not found"
//	@Failure	500	{object}	response.Problem
//	@Router		/api/v1/products/{id} [get]
func (h *ProductHandler) Get(c *gin.Context) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	p, err := h.products.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewProductResponse(&p))
}

// Create godoc
//
//	@Summary		Create product
//	@Description	SKU is normalized to upper case and must be unique among active products.
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.ProductRequest	true	"Product"
//	@Success		201		{object}	dto.ProductResponse
//	@Failure		400		{object}	response.Problem	"Malformed JSON or unknown field"
//	@Failure		404		{object}	response.Problem	"Category not found"
//	@Failure		409		{object}	response.Problem	"SKU already exists"
//	@Failure		413		{object}	response.Problem	"Body too large"
//	@Failure		415		{object}	response.Problem	"Content-Type is not application/json"
//	@Failure		422		{object}	response.Problem	"Validation failed"
//	@Failure		500		{object}	response.Problem
//	@Security		BearerAuth
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		403	{object}	response.Problem	"Insufficient permissions"
//	@Router			/api/v1/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var req dto.ProductRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	in, err := req.ToInput()
	if err != nil {
		response.Error(c, err)
		return
	}

	p, err := h.products.Create(c.Request.Context(), in)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Location", "/api/v1/products/"+p.ID.String())
	c.JSON(http.StatusCreated, dto.NewProductResponse(&p))
}

// Update godoc
//
//	@Summary		Replace product
//	@Description	Full update. `version` must match the current product version, otherwise 409 version_conflict is returned.
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Product ID"	format(uuid)
//	@Param			body	body		dto.UpdateProductRequest	true	"Product"
//	@Success		200		{object}	dto.ProductResponse
//	@Failure		400		{object}	response.Problem	"Malformed JSON, unknown field or invalid ID"
//	@Failure		404		{object}	response.Problem	"Product or category not found"
//	@Failure		409		{object}	response.Problem	"SKU already exists or version conflict"
//	@Failure		413		{object}	response.Problem	"Body too large"
//	@Failure		415		{object}	response.Problem	"Content-Type is not application/json"
//	@Failure		422		{object}	response.Problem	"Validation failed"
//	@Failure		500		{object}	response.Problem
//	@Security		BearerAuth
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		403	{object}	response.Problem	"Insufficient permissions"
//	@Router			/api/v1/products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req dto.UpdateProductRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	in, version, err := req.ToInput()
	if err != nil {
		response.Error(c, err)
		return
	}

	p, err := h.products.Update(c.Request.Context(), id, version, in)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewProductResponse(&p))
}

// Delete godoc
//
//	@Summary		Delete product
//	@Description	Soft delete. Past orders keep their snapshot of the product.
//	@Tags			products
//	@Param			id	path	string	true	"Product ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	response.Problem	"Invalid ID"
//	@Failure		404	{object}	response.Problem	"Product not found"
//	@Failure		500	{object}	response.Problem
//	@Security		BearerAuth
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		403	{object}	response.Problem	"Insufficient permissions"
//	@Router			/api/v1/products/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.products.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
