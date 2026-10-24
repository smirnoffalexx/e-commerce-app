package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const MaxCategoryNameLen = 64

var (
	ErrCategoryNotFound = NewError(ErrNotFound, "category_not_found", "category not found")
	ErrCategoryExists   = NewError(ErrConflict, "category_exists", "category with this name already exists")
	ErrCategoryInUse    = NewError(ErrConflict, "category_in_use", "category has products and cannot be deleted")
)

type Category struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCategory(name string) (Category, error) {
	c := Category{ID: uuid.Must(uuid.NewV7())}
	if err := c.Rename(name); err != nil {
		return Category{}, err
	}
	return c, nil
}

func (c *Category) Rename(name string) error {
	var verr ValidationError

	name = strings.TrimSpace(name)
	slug := Slugify(name)

	switch {
	case name == "":
		verr.Add("name", CodeRequired, "name is required")
	case utf8.RuneCountInString(name) > MaxCategoryNameLen:
		verr.Add("name", CodeTooLong, "name must be at most 64 characters")
	case hasControlChars(name, false):
		verr.Add("name", CodeInvalidFormat, "name must not contain control characters")
	case slug == "":
		verr.Add("name", CodeInvalidFormat, "name must contain latin letters or digits")
	}

	if err := verr.Err(); err != nil {
		return err
	}

	c.Name = name
	c.Slug = slug
	return nil
}

func Slugify(s string) string {
	s = strings.ReplaceAll(strings.ToLower(s), "&", " and ")

	var b strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}

	return strings.TrimSuffix(b.String(), "-")
}
