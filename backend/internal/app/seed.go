package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

func seedDemoCatalog(ctx context.Context, log *slog.Logger, path string, products usecase.ProductRepository, importer *usecase.CatalogImportService) error {
	if path == "" {
		return nil
	}

	_, total, err := products.List(ctx, domain.ProductFilter{Sort: domain.SortNewest, Limit: 1})
	if err != nil {
		return err
	}
	if total > 0 {
		log.Info("demo catalog skipped, products already exist", slog.Int64("products", total))
		return nil
	}

	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	report, err := importer.Import(ctx, f, usecase.ImportOptions{Mode: usecase.ImportModeUpsert, CreateMissingCategories: true})
	if err != nil {
		return err
	}
	log.Info("demo catalog imported",
		slog.String("file", path),
		slog.Int("rows", report.TotalRows),
		slog.Int("created", report.Created),
		slog.Int("failed", report.Failed),
		slog.Int("duplicates", len(report.Duplicates)),
		slog.Any("categories_created", report.CreatedCategories),
	)
	return nil
}
