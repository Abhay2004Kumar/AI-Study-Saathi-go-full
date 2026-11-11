package database

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schemaSQL embeds the complete database schema directly into the Go binary.
// This allows the Go application to self-initialize any fresh PostgreSQL database
// (e.g., Aiven, Supabase, Neon, or local Docker) with zero external CLI tools.
//
//go:embed schema.sql
var schemaSQL string

// AutoMigrate applies the schema to the database if tables do not exist yet.
// All statements are idempotent (CREATE ... IF NOT EXISTS).
func AutoMigrate(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("Running database auto-migration...")

	_, err := pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("applying database schema: %w", err)
	}

	slog.Info("Database auto-migration completed successfully")
	return nil
}

