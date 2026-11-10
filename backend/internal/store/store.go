package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store { return &Store{db: db} }

// Querier lets methods run on the pool or inside a transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (interface{ RowsAffected() int64 }, error)
}

func (s *Store) Tx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, s.db, fn)
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	return err
}

func jsonb(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// buildUpdate turns a whitelisted map into "set a=$1, b=$2" with args.
func buildUpdate(cols map[string]any, allowed map[string]bool, startIdx int) (string, []any, error) {
	var parts []string
	var args []any
	i := startIdx
	for k, v := range cols {
		if !allowed[k] {
			return "", nil, httpx.BadRequest(fmt.Sprintf("Field %q cannot be updated.", k))
		}
		parts = append(parts, fmt.Sprintf("%s = $%d", k, i))
		args = append(args, v)
		i++
	}
	if len(parts) == 0 {
		return "", nil, httpx.BadRequest("No valid fields were provided.")
	}
	return strings.Join(parts, ", "), args, nil
}
