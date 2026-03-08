package pgnotify

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// Executor is an interface satisfied by *pgx.Conn, *pgxpool.Pool, pgx.Tx, and similar types.
type Executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Notify sends a JSON-encoded payload on the specified PostgreSQL NOTIFY channel.
func Notify(ctx context.Context, exec Executor, channel string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("pgnotify: marshal payload: %w", err)
	}

	if _, err = exec.Exec(ctx, "SELECT pg_notify($1, $2)", channel, string(data)); err != nil {
		return fmt.Errorf("pgnotify: pg_notify: %w", err)
	}

	return nil
}
