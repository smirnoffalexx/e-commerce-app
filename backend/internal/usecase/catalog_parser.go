package usecase

import (
	"io"

	"e-commerce-backend/internal/domain"
)

type CatalogParser interface {
	Parse(r io.Reader) (domain.ParsedCatalog, error)
}
