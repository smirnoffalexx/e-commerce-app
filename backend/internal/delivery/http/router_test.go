package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	httpdelivery "e-commerce-backend/internal/delivery/http"
	"e-commerce-backend/internal/delivery/http/handler"
	"e-commerce-backend/internal/delivery/http/handler/mocks"
	"e-commerce-backend/internal/delivery/http/response"
	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type testAPI struct {
	router     *gin.Engine
	products   *mocks.ProductService
	categories *mocks.CategoryService
	auth       *mocks.AuthService
}

type stubTokens struct{}

func (stubTokens) Parse(token string) (domain.Principal, error) {
	principals := map[string]domain.Principal{
		"admin":      {UserID: uuid.MustParse("0192f6b8-0000-7000-8000-000000000001"), Role: domain.RoleAdmin},
		"superadmin": {UserID: uuid.MustParse("0192f6b8-0000-7000-8000-000000000002"), Role: domain.RoleSuperAdmin},
		"user":       {UserID: uuid.MustParse("0192f6b8-0000-7000-8000-000000000003"), Role: domain.RoleUser},
		"temp-admin": {UserID: uuid.MustParse("0192f6b8-0000-7000-8000-000000000004"), Role: domain.RoleAdmin, MustChangePassword: true},
	}
	p, ok := principals[token]
	if !ok {
		return domain.Principal{}, domain.ErrInvalidToken
	}
	return p, nil
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	products := mocks.NewProductService(t)
	categories := mocks.NewCategoryService(t)
	authSvc := mocks.NewAuthService(t)
	router := httpdelivery.NewRouter(httpdelivery.RouterDeps{
		Logger:     slog.New(slog.DiscardHandler),
		Health:     handler.NewHealthHandler(time.Second),
		Products:   handler.NewProductHandler(products),
		Categories: handler.NewCategoryHandler(categories),
		Auth:       handler.NewAuthHandler(authSvc),
		Tokens:     stubTokens{},
	})
	return &testAPI{router: router, products: products, categories: categories, auth: authSvc}
}

func (a *testAPI) do(method, path, body string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer admin")
	for i := 0; i+1 < len(headers); i += 2 {
		if headers[i+1] == "" {
			req.Header.Del(headers[i])
			continue
		}
		req.Header.Set(headers[i], headers[i+1])
	}
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, req)
	return w
}

func decodeProblem(t *testing.T, w *httptest.ResponseRecorder) response.Problem {
	t.Helper()
	assert.Equal(t, response.ContentTypeProblem, w.Header().Get("Content-Type"))
	var p response.Problem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &p), w.Body.String())
	assert.Equal(t, w.Code, p.Status)
	assert.NotEmpty(t, p.RequestID)
	return p
}

func fieldCodes(p response.Problem) map[string]string {
	out := make(map[string]string, len(p.Errors))
	for _, e := range p.Errors {
		out[e.Field] = e.Code
	}
	return out
}

const validProduct = `{"sku":"rs-001","name":"Running Shoes","description":"","category_id":"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b","price":"89.99","stock":150,"weight_kg":0.35}`

func TestCreateProduct_RejectsBadRequestsWithoutCallingService(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		headers    []string
		wantStatus int
		wantCode   string
		wantFields map[string]string
	}{
		{
			name:       "malformed json",
			body:       `{"sku": "RS-001",`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "malformed_json",
		},
		{
			name:       "empty body",
			body:       "",
			headers:    []string{"Content-Type", "application/json"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "empty_body",
		},
		{
			name:       "wrong content type",
			body:       validProduct,
			headers:    []string{"Content-Type", "text/plain"},
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "unsupported_media_type",
		},
		{
			name:       "unknown field",
			body:       `{"sku":"RS-001","colour":"red"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "unknown_field",
			wantFields: map[string]string{"colour": "unknown_field"},
		},
		{
			name:       "wrong field type",
			body:       `{"sku":"RS-001","stock":"many"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_type",
			wantFields: map[string]string{"stock": "invalid_type"},
		},
		{
			name:       "array instead of object",
			body:       `[]`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_type",
		},
		{
			name:       "trailing data",
			body:       validProduct + `{}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "malformed_json",
		},
		{
			name:       "body too large",
			body:       `{"description":"` + strings.Repeat("a", response.MaxJSONBodyBytes) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   "body_too_large",
		},
		{
			name:       "all invalid fields reported at once",
			body:       `{"sku":"","name":"   ","category_id":"not-a-uuid","price":"free","stock":-5}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_failed",
			wantFields: map[string]string{
				"sku":         domain.CodeRequired,
				"name":        domain.CodeRequired,
				"category_id": domain.CodeInvalidFormat,
				"price":       domain.CodeInvalidFormat,
				"stock":       domain.CodeNegative,
				"weight_kg":   domain.CodeRequired,
			},
		},
		{
			name:       "null price is treated as missing",
			body:       `{"sku":"RS-001","name":"Shoes","category_id":"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b","price":null,"stock":1,"weight_kg":"1"}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_failed",
			wantFields: map[string]string{"price": domain.CodeRequired},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := newTestAPI(t)

			w := api.do(http.MethodPost, "/api/v1/products", tc.body, tc.headers...)

			require.Equal(t, tc.wantStatus, w.Code, w.Body.String())
			p := decodeProblem(t, w)
			assert.Equal(t, tc.wantCode, p.Code)
			if tc.wantFields != nil {
				assert.Equal(t, tc.wantFields, fieldCodes(p))
			}
		})
	}
}

func TestCreateProduct_Success(t *testing.T) {
	api := newTestAPI(t)
	api.products.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in domain.ProductInput) bool {
		return in.Price.Equal(decimal.RequireFromString("89.99")) && in.WeightKg.Equal(decimal.RequireFromString("0.35")) && in.Stock == 150
	})).RunAndReturn(func(_ context.Context, in domain.ProductInput) (domain.Product, error) {
		return domain.NewProduct(in)
	})

	w := api.do(http.MethodPost, "/api/v1/products", validProduct)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "RS-001", body["sku"])
	assert.Equal(t, "89.99", body["price"], "money is serialized as a string")
	id, ok := body["id"].(string)
	require.True(t, ok)
	assert.Equal(t, "/api/v1/products/"+id, w.Header().Get("Location"))
}

func TestProductErrors_AreMappedToStatusCodes(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", domain.ErrProductNotFound, http.StatusNotFound, "product_not_found"},
		{"category missing", domain.ErrCategoryNotFound, http.StatusNotFound, "category_not_found"},
		{"sku taken", domain.ErrSKUTaken, http.StatusConflict, "sku_taken"},
		{"version conflict", domain.ErrProductVersionStale, http.StatusConflict, "version_conflict"},
		{"plain conflict kind", domain.ErrConflict, http.StatusConflict, "conflict"},
		{"unexpected failure", errors.New("pq: connection reset by peer"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := newTestAPI(t)
			api.products.EXPECT().Update(mock.Anything, id, 3, mock.Anything).Return(domain.Product{}, tc.err)

			body := strings.Replace(validProduct, `{`, `{"version":3,`, 1)
			w := api.do(http.MethodPut, "/api/v1/products/"+id.String(), body)

			require.Equal(t, tc.wantStatus, w.Code)
			p := decodeProblem(t, w)
			assert.Equal(t, tc.wantCode, p.Code)
			assert.NotContains(t, w.Body.String(), "pq:", "internal details must not leak")
		})
	}
}

func TestUpdateProduct_RequiresVersion(t *testing.T) {
	api := newTestAPI(t)

	w := api.do(http.MethodPut, "/api/v1/products/"+uuid.NewString(), validProduct)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, map[string]string{"version": domain.CodeRequired}, fieldCodes(decodeProblem(t, w)))
}

func TestPathAndQueryErrors(t *testing.T) {
	t.Run("invalid product id", func(t *testing.T) {
		api := newTestAPI(t)
		w := api.do(http.MethodGet, "/api/v1/products/42", "")
		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "invalid_path_param", decodeProblem(t, w).Code)
	})

	t.Run("malformed query params", func(t *testing.T) {
		api := newTestAPI(t)
		w := api.do(http.MethodGet, "/api/v1/products?page=abc&category_id=xyz", "")
		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, map[string]string{"page": domain.CodeInvalidFormat, "category_id": domain.CodeInvalidFormat}, fieldCodes(decodeProblem(t, w)))
	})

	t.Run("paging out of range from service", func(t *testing.T) {
		api := newTestAPI(t)
		verr := &domain.ValidationError{}
		verr.Add("page_size", domain.CodeInvalidFormat, "page_size must be between 1 and 100")
		api.products.EXPECT().List(mock.Anything, usecase.ListProductsQuery{PageRequest: usecase.PageRequest{PageSize: 1000}}).
			Return(usecase.Page[domain.Product]{}, verr)

		w := api.do(http.MethodGet, "/api/v1/products?page_size=1000", "")
		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Equal(t, map[string]string{"page_size": domain.CodeInvalidFormat}, fieldCodes(decodeProblem(t, w)))
	})
}

func TestRouterFallbacks(t *testing.T) {
	t.Run("unknown route", func(t *testing.T) {
		w := newTestAPI(t).do(http.MethodGet, "/api/v1/nope", "")
		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "route_not_found", decodeProblem(t, w).Code)
	})

	t.Run("method not allowed", func(t *testing.T) {
		w := newTestAPI(t).do(http.MethodPatch, "/api/v1/products", "")
		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
		assert.Equal(t, "method_not_allowed", decodeProblem(t, w).Code)
	})

	t.Run("panic becomes problem response", func(t *testing.T) {
		api := newTestAPI(t)
		api.categories.EXPECT().List(mock.Anything).RunAndReturn(func(context.Context) ([]domain.Category, error) {
			panic("boom")
		})

		w := api.do(http.MethodGet, "/api/v1/categories", "")
		require.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "internal_error", decodeProblem(t, w).Code)
		assert.NotContains(t, w.Body.String(), "boom")
	})
}

func TestWriteEndpoints_RequireAdmin(t *testing.T) {
	id := uuid.NewString()
	writes := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/products"},
		{http.MethodPut, "/api/v1/products/" + id},
		{http.MethodDelete, "/api/v1/products/" + id},
		{http.MethodPost, "/api/v1/categories"},
		{http.MethodPut, "/api/v1/categories/" + id},
		{http.MethodDelete, "/api/v1/categories/" + id},
	}
	cases := []struct {
		name       string
		auth       string
		wantStatus int
		wantCode   string
	}{
		{"no token", "", http.StatusUnauthorized, "missing_token"},
		{"malformed header", "Token admin", http.StatusUnauthorized, "missing_token"},
		{"invalid token", "Bearer forged", http.StatusUnauthorized, "invalid_token"},
		{"customer", "Bearer user", http.StatusForbidden, "insufficient_permissions"},
		{"admin with temporary password", "Bearer temp-admin", http.StatusForbidden, "password_change_required"},
	}

	for _, w := range writes {
		for _, tc := range cases {
			t.Run(w.method+" "+w.path+" "+tc.name, func(t *testing.T) {
				api := newTestAPI(t)

				resp := api.do(w.method, w.path, `{}`, "Authorization", tc.auth)

				require.Equal(t, tc.wantStatus, resp.Code, resp.Body.String())
				assert.Equal(t, tc.wantCode, decodeProblem(t, resp).Code)
			})
		}
	}
}

func TestReadEndpoints_ArePublic(t *testing.T) {
	api := newTestAPI(t)
	api.categories.EXPECT().List(mock.Anything).Return([]domain.Category{}, nil)

	w := api.do(http.MethodGet, "/api/v1/categories", "", "Authorization", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSuperAdminCanWriteProducts(t *testing.T) {
	api := newTestAPI(t)
	id := uuid.Must(uuid.NewV7())
	api.products.EXPECT().Delete(mock.Anything, id).Return(nil)

	w := api.do(http.MethodDelete, "/api/v1/products/"+id.String(), "", "Authorization", "Bearer superadmin")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestTemporaryPasswordAllowsOnlyAccountEndpoints(t *testing.T) {
	api := newTestAPI(t)
	tempID := uuid.MustParse("0192f6b8-0000-7000-8000-000000000004")
	api.auth.EXPECT().Me(mock.Anything, tempID).Return(domain.User{ID: tempID, Role: domain.RoleAdmin, MustChangePassword: true}, nil)

	w := api.do(http.MethodGet, "/api/v1/auth/me", "", "Authorization", "Bearer temp-admin")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"must_change_password":true`)
}

func TestLogin_InvalidCredentialsIs401(t *testing.T) {
	api := newTestAPI(t)
	api.auth.EXPECT().Login(mock.Anything, "jane@example.com", "wrong-password").Return(usecase.Session{}, domain.ErrInvalidCredentials)

	w := api.do(http.MethodPost, "/api/v1/auth/login", `{"email":"jane@example.com","password":"wrong-password"}`, "Authorization", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "invalid_credentials", decodeProblem(t, w).Code)
}
