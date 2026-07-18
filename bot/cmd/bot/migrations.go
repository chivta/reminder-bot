package main

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"

	"github.com/chivta/reminder-bot/internal/migrations"
)

// runMigrations applies all pending tern migrations using a dedicated
// connection (tern's migrator requires a *pgx.Conn, not a pool).
func runMigrations(ctx context.Context, conn *pgx.Conn) error {
	m, err := migrate.NewMigrator(ctx, conn, "public.schema_version")
	if err != nil {
		return err
	}
	if err := m.LoadMigrations(migrations.FS); err != nil {
		return err
	}
	return m.Migrate(ctx)
}
