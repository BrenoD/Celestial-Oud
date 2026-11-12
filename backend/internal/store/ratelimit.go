package store

import (
	"context"
	"time"
)

// Hit counts one request against key in a fixed window shared by every API
// instance (unlogged table, one upsert). It returns whether the request is
// within limit and how long until the window resets.
func (s *Store) Hit(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	var hits int
	var start time.Time
	err := s.db.QueryRow(ctx, `insert into rate_limits (key, window_start, hits) values ($1, now(), 1)
		on conflict (key) do update set
		  hits = case when rate_limits.window_start <= now() - make_interval(secs => $2) then 1 else rate_limits.hits + 1 end,
		  window_start = case when rate_limits.window_start <= now() - make_interval(secs => $2) then now() else rate_limits.window_start end
		returning hits, window_start`, key, window.Seconds()).Scan(&hits, &start)
	if err != nil {
		return false, 0, err
	}
	return hits <= limit, time.Until(start.Add(window)), nil
}

// Hits reads the current count for key without adding to it.
func (s *Store) Hits(ctx context.Context, key string, window time.Duration) int {
	var hits int
	_ = s.db.QueryRow(ctx, `select hits from rate_limits where key = $1 and window_start > now() - make_interval(secs => $2)`,
		key, window.Seconds()).Scan(&hits)
	return hits
}

// ClearHits resets a counter (e.g. failed logins after a successful one).
func (s *Store) ClearHits(ctx context.Context, key string) {
	_, _ = s.db.Exec(ctx, `delete from rate_limits where key = $1`, key)
}
