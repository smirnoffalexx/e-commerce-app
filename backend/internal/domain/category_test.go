package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Electronics":     "electronics",
		"Food & Beverage": "food-and-beverage",
		"Home & Office":   "home-and-office",
		"  Misc  ":        "misc",
		"Kids/Toys 3+":    "kids-toys-3",
		"--Weird__Name--": "weird-name",
		"Кухня":           "",
		"Кухня Kitchen 2": "kitchen-2",
	}
	for in, want := range cases {
		assert.Equal(t, want, Slugify(in), in)
	}
}

func TestNewCategory(t *testing.T) {
	c, err := NewCategory("  Food & Beverage ")
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, c.ID)
	assert.Equal(t, "Food & Beverage", c.Name)
	assert.Equal(t, "food-and-beverage", c.Slug)
}

func TestNewCategory_Invalid(t *testing.T) {
	cases := map[string]struct {
		name string
		code string
	}{
		"empty":              {"", CodeRequired},
		"whitespace only":    {"   ", CodeRequired},
		"too long":           {strings.Repeat("a", MaxCategoryNameLen+1), CodeTooLong},
		"control characters": {"Toys\x00", CodeInvalidFormat},
		"no latin or digits": {"Кухня", CodeInvalidFormat},
	}

	for title, tc := range cases {
		t.Run(title, func(t *testing.T) {
			_, err := NewCategory(tc.name)

			var verr *ValidationError
			require.ErrorAs(t, err, &verr)
			require.Len(t, verr.Fields, 1)
			assert.Equal(t, "name", verr.Fields[0].Field)
			assert.Equal(t, tc.code, verr.Fields[0].Code)
		})
	}
}

func TestCategory_RenameKeepsStateOnError(t *testing.T) {
	c, err := NewCategory("Games")
	require.NoError(t, err)

	require.ErrorIs(t, c.Rename(""), ErrValidation)
	assert.Equal(t, "Games", c.Name)
	assert.Equal(t, "games", c.Slug)

	require.NoError(t, c.Rename("Board Games"))
	assert.Equal(t, "board-games", c.Slug)
}
