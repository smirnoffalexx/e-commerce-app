package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

type orderModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID         uuid.UUID `gorm:"type:uuid"`
	CustomerEmail  string
	Status         string
	Total          decimal.Decimal `gorm:"type:numeric(16,2)"`
	IdempotencyKey string
	RequestHash    string
	CardLast4      string `gorm:"column:card_last4"`
	PaymentID      *string
	FailureCode    *string
	PaidAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Items          []orderItemModel `gorm:"foreignKey:OrderID"`
}

func (orderModel) TableName() string { return "orders" }

type orderItemModel struct {
	OrderID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	LineNo    int       `gorm:"primaryKey"`
	ProductID uuid.UUID `gorm:"type:uuid"`
	SKU       string    `gorm:"column:sku"`
	Name      string
	UnitPrice decimal.Decimal `gorm:"type:numeric(12,2)"`
	Quantity  int
	LineTotal decimal.Decimal `gorm:"type:numeric(16,2)"`
}

func (orderItemModel) TableName() string { return "order_items" }

func orderToModel(o *domain.Order) orderModel {
	m := orderModel{
		ID:             o.ID,
		UserID:         o.UserID,
		CustomerEmail:  o.CustomerEmail,
		Status:         string(o.Status),
		Total:          o.Total,
		IdempotencyKey: o.IdempotencyKey,
		RequestHash:    o.RequestHash,
		CardLast4:      o.CardLast4,
		PaymentID:      optional(o.PaymentID),
		FailureCode:    optional(o.FailureCode),
		PaidAt:         o.PaidAt,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
		Items:          make([]orderItemModel, 0, len(o.Items)),
	}
	for i, it := range o.Items {
		m.Items = append(m.Items, orderItemModel{
			OrderID:   o.ID,
			LineNo:    i + 1,
			ProductID: it.ProductID,
			SKU:       it.SKU,
			Name:      it.Name,
			UnitPrice: it.UnitPrice,
			Quantity:  it.Quantity,
			LineTotal: it.LineTotal,
		})
	}
	return m
}

func orderToDomain(m *orderModel) domain.Order {
	o := domain.Order{
		ID:             m.ID,
		UserID:         m.UserID,
		CustomerEmail:  m.CustomerEmail,
		Status:         domain.OrderStatus(m.Status),
		Total:          m.Total,
		IdempotencyKey: m.IdempotencyKey,
		RequestHash:    m.RequestHash,
		CardLast4:      m.CardLast4,
		CreatedAt:      m.CreatedAt.UTC(),
		UpdatedAt:      m.UpdatedAt.UTC(),
		Items:          make([]domain.OrderItem, 0, len(m.Items)),
	}
	if m.PaymentID != nil {
		o.PaymentID = *m.PaymentID
	}
	if m.FailureCode != nil {
		o.FailureCode = *m.FailureCode
	}
	if m.PaidAt != nil {
		t := m.PaidAt.UTC()
		o.PaidAt = &t
	}
	for _, it := range m.Items {
		o.Items = append(o.Items, domain.OrderItem{
			ProductID: it.ProductID,
			SKU:       it.SKU,
			Name:      it.Name,
			UnitPrice: it.UnitPrice,
			Quantity:  it.Quantity,
			LineTotal: it.LineTotal,
		})
	}
	return o
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) LockProducts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.Product, error) {
	var models []productModel
	err := conn(ctx, r.db).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id IN ?", ids).
		Order("id").
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("lock products: %w", err)
	}
	result := make(map[uuid.UUID]domain.Product, len(models))
	for i := range models {
		result[models[i].ID] = productToDomain(&models[i])
	}
	return result, nil
}

func (r *OrderRepository) ReserveStock(ctx context.Context, items []domain.OrderItem) error {
	db := conn(ctx, r.db)
	for _, it := range items {
		res := db.Model(&productModel{}).
			Where("id = ? AND stock - reserved >= ?", it.ProductID, it.Quantity).
			Updates(map[string]any{"reserved": gorm.Expr("reserved + ?", it.Quantity)})
		if res.Error != nil {
			return fmt.Errorf("reserve stock: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return &domain.CheckoutConflictError{Issues: []domain.CheckoutIssue{{
				ProductID:         it.ProductID,
				Code:              domain.IssueInsufficientStock,
				RequestedQuantity: it.Quantity,
			}}}
		}
	}
	return nil
}

func (r *OrderRepository) ReleaseStock(ctx context.Context, items []domain.OrderItem) error {
	return r.settleReserve(ctx, items, false)
}

func (r *OrderRepository) CommitStock(ctx context.Context, items []domain.OrderItem) error {
	return r.settleReserve(ctx, items, true)
}

// settleReserve drops the reservation of each item. On commit the units also leave
// stock and the version is bumped, so an admin edit based on the old stock is stale.
func (r *OrderRepository) settleReserve(ctx context.Context, items []domain.OrderItem, commit bool) error {
	db := conn(ctx, r.db).Unscoped()
	for _, it := range items {
		changes := map[string]any{"reserved": gorm.Expr("reserved - ?", it.Quantity)}
		if commit {
			changes["stock"] = gorm.Expr("stock - ?", it.Quantity)
			changes["version"] = gorm.Expr("version + 1")
			changes["updated_at"] = gorm.Expr("now()")
		}
		if err := db.Model(&productModel{}).Where("id = ?", it.ProductID).Updates(changes).Error; err != nil {
			return fmt.Errorf("settle reserved stock: %w", err)
		}
	}
	return nil
}

func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	m := orderToModel(o)
	return conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return domain.ErrIdempotencyKeyTaken
			}
			return fmt.Errorf("create order: %w", err)
		}
		if err := tx.Create(&m.Items).Error; err != nil {
			return fmt.Errorf("create order items: %w", err)
		}
		*o = orderToDomain(&m)
		return nil
	})
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	return r.get(conn(ctx, r.db), "id = ?", id)
}

func (r *OrderRepository) GetByIdempotencyKey(ctx context.Context, userID uuid.UUID, key string) (domain.Order, error) {
	return r.get(conn(ctx, r.db), "user_id = ? AND idempotency_key = ?", userID, key)
}

func (r *OrderRepository) LockByID(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	return r.get(conn(ctx, r.db).Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}), "id = ?", id)
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, o *domain.Order) error {
	m := orderToModel(o)
	now := r.db.NowFunc()
	res := conn(ctx, r.db).
		Model(&orderModel{}).
		Where("id = ? AND status = ?", o.ID, string(domain.OrderPending)).
		Updates(map[string]any{
			"status":       m.Status,
			"payment_id":   m.PaymentID,
			"failure_code": m.FailureCode,
			"paid_at":      m.PaidAt,
			"updated_at":   now,
		})
	if res.Error != nil {
		return fmt.Errorf("update order status: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrOrderNotPending
	}
	o.UpdatedAt = now
	return nil
}

func (r *OrderRepository) List(ctx context.Context, f domain.OrderFilter) ([]domain.Order, int64, error) {
	q := conn(ctx, r.db).Model(&orderModel{})
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	var models []orderModel
	err := withItems(q).Order("created_at DESC, id DESC").Limit(f.Limit).Offset(f.Offset).Find(&models).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}

	orders := make([]domain.Order, 0, len(models))
	for i := range models {
		orders = append(orders, orderToDomain(&models[i]))
	}
	return orders, total, nil
}

func (r *OrderRepository) get(q *gorm.DB, query string, args ...any) (domain.Order, error) {
	var m orderModel
	if err := withItems(q).Where(query, args...).Take(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Order{}, domain.ErrOrderNotFound
		}
		return domain.Order{}, fmt.Errorf("get order: %w", err)
	}
	return orderToDomain(&m), nil
}

func withItems(q *gorm.DB) *gorm.DB {
	return q.Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("line_no") })
}

var _ usecase.OrderRepository = (*OrderRepository)(nil)
