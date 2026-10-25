package handler

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/dto"
	"e-commerce-backend/internal/delivery/http/response"
	"e-commerce-backend/internal/domain"
)

const (
	MaxImportFileBytes   = 5 << 20
	multipartOverhead    = 64 << 10
	multipartMemoryBytes = 1 << 20
)

type CatalogImportHandler struct {
	imports CatalogImportService
}

func NewCatalogImportHandler(imports CatalogImportService) *CatalogImportHandler {
	return &CatalogImportHandler{imports: imports}
}

func (h *CatalogImportHandler) Register(r gin.IRouter, guard ...gin.HandlerFunc) {
	r.POST("/products/import", append(append([]gin.HandlerFunc{}, guard...), h.Import)...)
}

// Import godoc
//
//	@Summary		Import products from CSV
//	@Description	Upload a CSV with columns name, sku, description, category, price, stock, weight_kg (UTF-8, header row required, max 5 MB / 10000 rows). Every row is validated with the same rules as the product API. Duplicate SKUs inside the file: the last row wins and the earlier lines are listed in `duplicates`. Existing SKUs are handled by `mode`: upsert updates them, skip leaves them untouched, fail reports them as errors. With `atomic=true` nothing is written if any row fails. With `dry_run=true` the import runs inside a transaction that is rolled back, so the report is exact but nothing is saved. The response is 200 whenever the file itself could be processed; check `committed` and `errors`.
//	@Tags			products
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file						formData	file	true	"CSV file"
//	@Param			mode						query		string	false	"How to treat SKUs that already exist"	Enums(upsert,skip,fail)	default(upsert)
//	@Param			dry_run						query		bool	false	"Validate and report without saving"
//	@Param			atomic						query		bool	false	"Save nothing if any row fails"
//	@Param			create_missing_categories	query		bool	false	"Create categories that do not exist yet"
//	@Success		200							{object}	dto.ImportReportResponse
//	@Failure		400							{object}	response.Problem	"Missing file or malformed query parameters"
//	@Failure		401							{object}	response.Problem	"Missing or invalid token"
//	@Failure		403							{object}	response.Problem	"Insufficient permissions"
//	@Failure		409							{object}	response.Problem	"SKU created concurrently, retry"
//	@Failure		413							{object}	response.Problem	"File larger than 5 MB"
//	@Failure		415							{object}	response.Problem	"Not multipart/form-data"
//	@Failure		422							{object}	response.Problem	"Invalid header, too many rows or invalid mode"
//	@Failure		500							{object}	response.Problem
//	@Router			/api/v1/products/import [post]
func (h *CatalogImportHandler) Import(c *gin.Context) {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		response.Error(c, &response.RequestError{
			Status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type", Detail: "Content-Type must be multipart/form-data",
		})
		return
	}

	q := response.NewQueryParser(c)
	opts := domain.ImportOptions{
		Mode:                    domain.ImportMode(c.Query("mode")),
		DryRun:                  q.Bool("dry_run"),
		Atomic:                  q.Bool("atomic"),
		CreateMissingCategories: q.Bool("create_missing_categories"),
	}
	if err := q.Err(); err != nil {
		response.Error(c, err)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxImportFileBytes+multipartOverhead)
	if err := c.Request.ParseMultipartForm(multipartMemoryBytes); err != nil {
		response.Error(c, uploadError(err))
		return
	}
	defer func() { _ = c.Request.MultipartForm.RemoveAll() }()

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, fileFieldError(http.StatusBadRequest, domain.CodeRequired, "multipart field \"file\" with a CSV file is required"))
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > MaxImportFileBytes {
		response.Error(c, tooLarge())
		return
	}
	if ext := strings.ToLower(filepath.Ext(header.Filename)); ext != ".csv" {
		response.Error(c, fileFieldError(http.StatusUnprocessableEntity, domain.CodeInvalidFormat, "file must have a .csv extension"))
		return
	}

	report, err := h.imports.Import(c.Request.Context(), file, opts)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewImportReportResponse(&report))
}

func uploadError(err error) error {
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		return tooLarge()
	}
	return &response.RequestError{Status: http.StatusBadRequest, Code: "malformed_multipart", Detail: "request body is not a valid multipart form"}
}

func tooLarge() error {
	return &response.RequestError{
		Status: http.StatusRequestEntityTooLarge, Code: "file_too_large", Detail: fmt.Sprintf("file must not exceed %d bytes", MaxImportFileBytes),
	}
}

func fileFieldError(status int, code, msg string) error {
	return &response.RequestError{
		Status: status, Code: code, Detail: msg,
		Fields: []domain.FieldError{{Field: "file", Code: code, Message: msg}},
	}
}
