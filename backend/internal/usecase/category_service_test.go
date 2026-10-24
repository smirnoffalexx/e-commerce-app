package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
	"e-commerce-backend/internal/usecase/mocks"
)

func TestCategoryService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("stores category with slug", func(t *testing.T) {
		repo := mocks.NewCategoryRepository(t)
		repo.EXPECT().Create(ctx, mock.MatchedBy(func(c *domain.Category) bool {
			return c.Name == "Food & Beverage" && c.Slug == "food-and-beverage"
		})).Return(nil)

		c, err := usecase.NewCategoryService(repo).Create(ctx, " Food & Beverage ")
		require.NoError(t, err)
		assert.Equal(t, "food-and-beverage", c.Slug)
	})

	t.Run("invalid name never reaches repository", func(t *testing.T) {
		repo := mocks.NewCategoryRepository(t)

		_, err := usecase.NewCategoryService(repo).Create(ctx, "   ")
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("duplicate is a conflict", func(t *testing.T) {
		repo := mocks.NewCategoryRepository(t)
		repo.EXPECT().Create(ctx, mock.Anything).Return(domain.ErrCategoryExists)

		_, err := usecase.NewCategoryService(repo).Create(ctx, "Games")
		assert.ErrorIs(t, err, domain.ErrConflict)
	})
}

func TestCategoryService_Rename(t *testing.T) {
	ctx := context.Background()
	current, err := domain.NewCategory("Games")
	require.NoError(t, err)

	t.Run("renames and regenerates slug", func(t *testing.T) {
		repo := mocks.NewCategoryRepository(t)
		repo.EXPECT().GetByID(ctx, current.ID).Return(current, nil)
		repo.EXPECT().Update(ctx, mock.MatchedBy(func(c *domain.Category) bool {
			return c.ID == current.ID && c.Slug == "board-games"
		})).Return(nil)

		c, err := usecase.NewCategoryService(repo).Rename(ctx, current.ID, "Board Games")
		require.NoError(t, err)
		assert.Equal(t, "Board Games", c.Name)
	})

	t.Run("invalid name never reaches repository", func(t *testing.T) {
		repo := mocks.NewCategoryRepository(t)
		repo.EXPECT().GetByID(ctx, current.ID).Return(current, nil)

		_, err := usecase.NewCategoryService(repo).Rename(ctx, current.ID, "")
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("missing category", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		repo := mocks.NewCategoryRepository(t)
		repo.EXPECT().GetByID(ctx, id).Return(domain.Category{}, domain.ErrCategoryNotFound)

		_, err := usecase.NewCategoryService(repo).Rename(ctx, id, "New")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestCategoryService_Delete_InUse(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	repo := mocks.NewCategoryRepository(t)
	repo.EXPECT().Delete(mock.Anything, id).Return(domain.ErrCategoryInUse)

	err := usecase.NewCategoryService(repo).Delete(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrCategoryInUse)
}
