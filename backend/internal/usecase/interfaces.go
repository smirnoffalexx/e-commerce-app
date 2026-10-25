package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

type CategoryRepository interface {
	Create(ctx context.Context, c *domain.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Category, error)
	GetBySlugs(ctx context.Context, slugs []string) (map[string]domain.Category, error)
	List(ctx context.Context) ([]domain.Category, error)
	Update(ctx context.Context, c *domain.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
	List(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int64, error)
	Update(ctx context.Context, p *domain.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	ListByRole(ctx context.Context, role domain.Role, limit, offset int) ([]domain.User, int64, error)
	Update(ctx context.Context, u *domain.User) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

type AccessTokenIssuer interface {
	Issue(u *domain.User) (token string, expiresAt time.Time, err error)
}

type TempPasswordGenerator interface {
	Generate() (string, error)
}

type RefreshTokenStore interface {
	Issue(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error)
	Lookup(ctx context.Context, token string) (uuid.UUID, error)
	Consume(ctx context.Context, token string) (uuid.UUID, error)
	Revoke(ctx context.Context, token string) error
	RevokeAll(ctx context.Context, userID uuid.UUID) error
}

type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type OrderRepository interface {
	LockProducts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.Product, error)
	ReserveStock(ctx context.Context, items []domain.OrderItem) error
	ReleaseStock(ctx context.Context, items []domain.OrderItem) error
	CommitStock(ctx context.Context, items []domain.OrderItem) error
	Create(ctx context.Context, o *domain.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Order, error)
	GetByIdempotencyKey(ctx context.Context, userID uuid.UUID, key string) (domain.Order, error)
	LockByID(ctx context.Context, id uuid.UUID) (domain.Order, error)
	UpdateStatus(ctx context.Context, o *domain.Order) error
	List(ctx context.Context, f domain.OrderFilter) ([]domain.Order, int64, error)
}

type PaymentProvider interface {
	Charge(ctx context.Context, req domain.PaymentRequest) (domain.PaymentResult, error)
}

type CheckoutLocker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (token string, acquired bool, err error)
	Release(ctx context.Context, key, token string) error
}

type CatalogImportRepository interface {
	LockCatalogImport(ctx context.Context) error
	CategoriesBySlugs(ctx context.Context, slugs []string) (map[string]domain.Category, error)
	ProductsBySKUs(ctx context.Context, skus []string) (map[string]domain.Product, error)
	CreateProducts(ctx context.Context, products []domain.Product) error
}
