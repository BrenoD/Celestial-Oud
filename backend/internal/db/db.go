// Package db owns the pgx pool and the embedded SQL migration runner.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed all:migrations
var migrationFS embed.FS

type DB struct {
	*pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	// Containers scale horizontally; keep each instance's pool small so the
	// Supabase pooler / Hyperdrive is not exhausted.
	cfg.MaxConns = 8
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{pool}, nil
}

// Migrate applies every *.sql file in migrations/ that is not yet recorded in
// schema_migrations, in filename order, each inside its own transaction.
func (d *DB) Migrate(ctx context.Context, log *slog.Logger) error {
	if _, err := d.Exec(ctx, `create table if not exists schema_migrations (
		version text primary key, applied_at timestamptz not null default now())`); err != nil {
		return err
	}
	// Serialise concurrent instances starting at once.
	conn, err := d.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `select pg_advisory_lock(727001)`); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), `select pg_advisory_unlock(727001)`) //nolint:errcheck

	applied := map[string]bool{}
	rows, err := conn.Query(ctx, `select version from schema_migrations`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if applied[name] {
			continue
		}
		sqlBytes, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
			_, err := tx.Exec(ctx, `insert into schema_migrations(version) values($1)`, name)
			return err
		})
		if err != nil {
			return err
		}
		log.Info("migration applied", "version", name)
	}
	return nil
}
