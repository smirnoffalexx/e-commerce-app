//go:build integration

package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	ctx := context.Background()

	migrations, err := filepath.Glob(filepath.Join("..", "..", "..", "migrations", "*.up.sql"))
	if err != nil || len(migrations) == 0 {
		fmt.Fprintln(os.Stderr, "migrations not found", err)
		return 1
	}
	sort.Strings(migrations)

	container, err := tcpostgres.Run(ctx, "postgres:18.6-alpine",
		tcpostgres.WithDatabase("ecommerce_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.WithOrderedInitScripts(migrations...),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start postgres container:", err)
		return 1
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "connection string:", err)
		return 1
	}

	testDB, err = NewDB(ctx, dsn, slog.New(slog.DiscardHandler))
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		return 1
	}
	defer func() { _ = Close(testDB) }()

	return m.Run()
}

func resetDB(t *testing.T) {
	t.Helper()
	if err := testDB.Exec("TRUNCATE products, categories CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
