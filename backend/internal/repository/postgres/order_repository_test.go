//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/payment"
	"e-commerce-backend/internal/usecase"
)

type noopLocker struct{}

func (noopLocker) Acquire(context.Context, string, time.Duration) (string, bool, error) {
	return "token", true, nil
}

func (noopLocker) Release(context.Context, string, string) error { return nil }

func mustCustomer(t *testing.T, email string) domain.User {
	t.Helper()
	u, err := domain.NewUser(email, "hash", domain.RoleUser)
	require.NoError(t, err)
	require.NoError(t, NewUserRepository(testDB).Create(context.Background(), &u))
	return u
}

func checkoutFor(p domain.Product, quantity int, key, card string) domain.CheckoutInput {
	return domain.CheckoutInput{
		IdempotencyKey: key,
		Lines:          []domain.CheckoutLine{{ProductID: p.ID, Quantity: quantity, ExpectedPrice: p.Price}},
		Card:           domain.Card{Number: card, ExpMonth: 12, ExpYear: time.Now().Year() + 2, CVC: "123"},
	}
}

func newOrderService() *usecase.OrderService {
	return usecase.NewOrderService(NewTransactor(testDB), NewOrderRepository(testDB), NewUserRepository(testDB), payment.NewFakeProvider(), noopLocker{})
}

func TestOrderRepository(t *testing.T) {
	ctx := context.Background()
	orders := NewOrderRepository(testDB)
	products := NewProductRepository(testDB)
	categories := NewCategoryRepository(testDB)

	t.Run("stores snapshot, enforces idempotency key per user and guards status", func(t *testing.T) {
		resetDB(t)
		p := mustProduct(t, products, "RS-001", mustCategory(t, categories, "Shoes").ID)
		customer := mustCustomer(t, "jane@example.com")
		in := checkoutFor(p, 2, "k-1", "4242424242424242")

		o := domain.NewOrder(&customer, &in, map[uuid.UUID]domain.Product{p.ID: p})
		require.NoError(t, orders.Create(ctx, &o))

		got, err := orders.GetByIdempotencyKey(ctx, customer.ID, "k-1")
		require.NoError(t, err)
		assert.Equal(t, o.ID, got.ID)
		require.Len(t, got.Items, 1)
		assert.Equal(t, "RS-001", got.Items[0].SKU)
		assert.True(t, got.Total.Equal(decimal.RequireFromString("39.98")))

		dup := domain.NewOrder(&customer, &in, map[uuid.UUID]domain.Product{p.ID: p})
		assert.ErrorIs(t, orders.Create(ctx, &dup), domain.ErrIdempotencyKeyTaken)

		other := mustCustomer(t, "john@example.com")
		otherOrder := domain.NewOrder(&other, &in, map[uuid.UUID]domain.Product{p.ID: p})
		require.NoError(t, orders.Create(ctx, &otherOrder), "keys are scoped per user")

		require.NoError(t, got.MarkPaid("pay_1", time.Now()))
		require.NoError(t, orders.UpdateStatus(ctx, &got))
		assert.ErrorIs(t, orders.UpdateStatus(ctx, &got), domain.ErrOrderNotPending)

		paid, total, err := orders.List(ctx, domain.OrderFilter{UserID: &customer.ID, Status: domain.OrderPaid, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "pay_1", paid[0].PaymentID)
	})

	t.Run("reserve bumps version and never goes below zero, release restores", func(t *testing.T) {
		resetDB(t)
		p := mustProduct(t, products, "RS-001", mustCategory(t, categories, "Shoes").ID)
		items := []domain.OrderItem{{ProductID: p.ID, Quantity: 4}}

		require.NoError(t, orders.ReserveStock(ctx, items))
		after, err := products.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, 6, after.Stock)
		assert.Equal(t, p.Version+1, after.Version, "admin edits based on the old version must fail")

		var cerr *domain.CheckoutConflictError
		assert.ErrorAs(t, orders.ReserveStock(ctx, []domain.OrderItem{{ProductID: p.ID, Quantity: 7}}), &cerr)

		require.NoError(t, products.Delete(ctx, p.ID))
		require.NoError(t, orders.ReleaseStock(ctx, items), "stock returns even if the product was deleted meanwhile")
		var stock int
		require.NoError(t, testDB.Raw("SELECT stock FROM products WHERE id = ?", p.ID).Scan(&stock).Error)
		assert.Equal(t, 10, stock)
	})
}

func TestCheckout_EndToEnd(t *testing.T) {
	ctx := context.Background()
	products := NewProductRepository(testDB)
	categories := NewCategoryRepository(testDB)

	t.Run("declined payment returns stock and replay does not charge twice", func(t *testing.T) {
		resetDB(t)
		p := mustProduct(t, products, "RS-001", mustCategory(t, categories, "Shoes").ID)
		customer := mustCustomer(t, "jane@example.com")
		svc := newOrderService()

		res, err := svc.Checkout(ctx, customer.ID, checkoutFor(p, 3, "k-1", "4000000000000002"))
		require.NoError(t, err)
		assert.Equal(t, domain.OrderFailed, res.Order.Status)
		current, err := products.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, 10, current.Stock)

		res, err = svc.Checkout(ctx, customer.ID, checkoutFor(p, 3, "k-2", "4242424242424242"))
		require.NoError(t, err)
		assert.Equal(t, domain.OrderPaid, res.Order.Status)

		replay, err := svc.Checkout(ctx, customer.ID, checkoutFor(p, 3, "k-2", "4242424242424242"))
		require.NoError(t, err)
		assert.True(t, replay.Replayed)
		assert.Equal(t, res.Order.ID, replay.Order.ID)

		current, err = products.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, 7, current.Stock)
	})

	t.Run("many buyers, one last item: exactly one wins", func(t *testing.T) {
		resetDB(t)
		p := mustProduct(t, products, "RS-001", mustCategory(t, categories, "Shoes").ID)
		require.NoError(t, testDB.Exec("UPDATE products SET stock = 1 WHERE id = ?", p.ID).Error)
		svc := newOrderService()

		const buyers = 20
		customers := make([]domain.User, buyers)
		for i := range customers {
			customers[i] = mustCustomer(t, fmt.Sprintf("buyer%d@example.com", i))
		}

		var (
			wg        sync.WaitGroup
			mu        sync.Mutex
			paid      int
			conflicts int
		)
		start := make(chan struct{})
		for i := range buyers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				res, err := svc.Checkout(ctx, customers[i].ID, checkoutFor(p, 1, "last-item", "4242424242424242"))
				mu.Lock()
				defer mu.Unlock()
				var cerr *domain.CheckoutConflictError
				switch {
				case err == nil && res.Order.Status == domain.OrderPaid:
					paid++
				case errors.As(err, &cerr):
					conflicts++
				default:
					t.Errorf("unexpected result: %+v, %v", res.Order.Status, err)
				}
			}()
		}
		close(start)
		wg.Wait()

		assert.Equal(t, 1, paid)
		assert.Equal(t, buyers-1, conflicts)
		current, err := products.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Zero(t, current.Stock)
	})
}
