// Package db owns the pgx pool and the embedded SQL migration runner.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
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

	// Transaction-mode poolers (Supabase Supavisor on :6543, PgBouncer,
	// Cloudflare Hyperdrive) may run consecutive statements on different
	// backends, so cached named prepared statements break. DescribeExec uses
	// the unnamed statement each time: one extra round trip, but correct
	// parameter types (jsonb, arrays) and no server-side state.
	if UsesTransactionPooler(cfg.ConnConfig.Host, cfg.ConnConfig.Port) {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec
		cfg.ConnConfig.StatementCacheCapacity = 0
		cfg.ConnConfig.DescriptionCacheCapacity = 0
	}

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

// UsesTransactionPooler reports whether the target looks like a transaction-mode
// pooler. Override with DB_TRANSACTION_POOLER=true|false.
func UsesTransactionPooler(host string, port uint16) bool {
	switch strings.ToLower(os.Getenv("DB_TRANSACTION_POOLER")) {
	case "true", "1":
		return true
	case "false", "0":
		return false
	}
	return port == 6543 || strings.Contains(host, "pooler.supabase.com") || strings.Contains(host, "hyperdrive")
}

// Migrate applies every *.sql file in migrations/ that is not yet recorded in
// schema_migrations, in filename order, each inside its own transaction.
//
// Concurrency: instances may boot together. Each migration takes a
// transaction-scoped advisory lock and re-checks schema_migrations inside the
// same transaction, so it is safe behind transaction poolers (a session lock
// could stay stuck on a shared backend there).
func (d *DB) Migrate(ctx context.Context, log *slog.Logger) error {
	// "create table if not exists" is not atomic: two instances booting together
	// can both pass the check and one fails on pg_type. Serialise it with the
	// same transaction-scoped lock the migrations use.
	err := pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(727001)`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `create table if not exists schema_migrations (
			version text primary key, applied_at timestamptz not null default now())`)
		return err
	})
	if err != nil {
		return err
	}

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
		sqlBytes, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		applied := false
		err = pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(727001)`); err != nil {
				return err
			}
			var done bool
			if err := tx.QueryRow(ctx, `select exists(select 1 from schema_migrations where version = $1)`, name).Scan(&done); err != nil {
				return err
			}
			if done {
				return nil
			}
			// No arguments → pgx uses the simple protocol, which accepts
			// multi-statement SQL files.
			if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
			if _, err := tx.Exec(ctx, `insert into schema_migrations(version) values($1)`, name); err != nil {
				return err
			}
			applied = true
			return nil
		})
		if err != nil {
			return err
		}
		if applied {
			log.Info("migration applied", "version", name)
		}
	}
	return nil
}
