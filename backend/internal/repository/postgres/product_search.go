package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"e-commerce-backend/internal/domain"
)

const (
	wordSimilarityThreshold = "0.4"
	minFuzzyWordLen         = 3
	maxFuzzyWords           = 5

	exactCondition = `(
		products.search_vector @@ websearch_to_tsquery('english', ?)
		OR products.sku LIKE ? ESCAPE '\'
	)`

	exactRank = `
		CASE WHEN products.sku = ? THEN 10 ELSE 0 END
		+ CASE WHEN products.sku LIKE ? ESCAPE '\' THEN 2 ELSE 0 END
		+ ts_rank(products.search_vector, websearch_to_tsquery('english', ?)) * 4`
)

var sortOrders = map[domain.ProductSort]string{
	domain.SortRelevance: "rank DESC, products.created_at DESC, products.id DESC",
	domain.SortNewest:    "products.created_at DESC, products.id DESC",
	domain.SortPriceAsc:  "products.price ASC, products.id ASC",
	domain.SortPriceDesc: "products.price DESC, products.id DESC",
	domain.SortNameAsc:   "lower(products.name) ASC, products.id ASC",
}

func (r *ProductRepository) List(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int64, error) {
	order, ok := sortOrders[f.Sort]
	if !ok || (f.Sort == domain.SortRelevance && f.Query == "") {
		order = sortOrders[domain.SortNewest]
	}

	var (
		models []productModel
		total  int64
	)
	err := conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		search := exactSearch(f.Query)
		q := applyProductFilter(tx.Model(&productModel{}), &f, search)
		if err := q.Count(&total).Error; err != nil {
			return fmt.Errorf("count products: %w", err)
		}

		if total == 0 && f.Query != "" {
			search = fuzzySearch(f.Query)
			if search == nil {
				return nil
			}
			if err := tx.Exec("SET LOCAL pg_trgm.word_similarity_threshold = " + wordSimilarityThreshold).Error; err != nil {
				return fmt.Errorf("set similarity threshold: %w", err)
			}
			q = applyProductFilter(tx.Model(&productModel{}), &f, search)
			if err := q.Count(&total).Error; err != nil {
				return fmt.Errorf("count fuzzy products: %w", err)
			}
		}
		if total == 0 {
			return nil
		}

		find := r.withCategory(q)
		if search != nil {
			find = find.Select("products.*, ("+search.rank+") AS rank", search.rankArgs...)
		}
		if err := find.Order(order).Limit(f.Limit).Offset(f.Offset).Find(&models).Error; err != nil {
			return fmt.Errorf("list products: %w", err)
		}
		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, 0, err
	}

	products := make([]domain.Product, 0, len(models))
	for i := range models {
		products = append(products, productToDomain(&models[i]))
	}
	return products, total, nil
}

type textSearch struct {
	condition string
	args      []any
	rank      string
	rankArgs  []any
}

func exactSearch(query string) *textSearch {
	if query == "" {
		return nil
	}
	sku := strings.ToUpper(query)
	prefix := escapeLike(sku) + "%"
	return &textSearch{
		condition: exactCondition,
		args:      []any{query, prefix},
		rank:      exactRank,
		rankArgs:  []any{sku, prefix, query},
	}
}

func fuzzySearch(query string) *textSearch {
	var words []string
	for _, w := range strings.Fields(query) {
		if strings.HasPrefix(w, "-") {
			continue
		}
		w = strings.Trim(w, `"'+()`)
		if len([]rune(w)) >= minFuzzyWordLen {
			words = append(words, w)
		}
		if len(words) == maxFuzzyWords {
			break
		}
	}
	if len(words) == 0 {
		return nil
	}

	conds := make([]string, 0, len(words))
	ranks := make([]string, 0, len(words))
	args := make([]any, 0, len(words))
	for _, w := range words {
		conds = append(conds, "? <% products.name")
		ranks = append(ranks, "word_similarity(?, products.name)")
		args = append(args, w)
	}
	return &textSearch{
		condition: "(" + strings.Join(conds, " OR ") + ")",
		args:      args,
		rank:      strings.Join(ranks, " + "),
		rankArgs:  args,
	}
}

func applyProductFilter(q *gorm.DB, f *domain.ProductFilter, search *textSearch) *gorm.DB {
	if search != nil {
		q = q.Where(search.condition, search.args...)
	}
	if f.CategoryID != nil {
		q = q.Where("products.category_id = ?", *f.CategoryID)
	}
	if f.MinPrice != nil {
		q = q.Where("products.price >= ?", *f.MinPrice)
	}
	if f.MaxPrice != nil {
		q = q.Where("products.price <= ?", *f.MaxPrice)
	}
	if f.InStock {
		q = q.Where("products.stock > 0")
	}
	return q
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
