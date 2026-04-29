package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"

	_ "github.com/lib/pq"
)

// ApplyMigrationsFromConfig opens a short-lived database connection and applies
// the embedded SQL migrations. It is used by container entrypoints before the
// main server process starts.
func ApplyMigrationsFromConfig(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}
	if err := timezone.Init(cfg.Timezone); err != nil {
		return err
	}

	db, err := sql.Open("postgres", cfg.Database.DSNWithTimezone(cfg.Timezone))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	applyDBPoolSettings(db, cfg)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if err := ApplyMigrations(ctx, db); err != nil {
		return err
	}
	return nil
}
