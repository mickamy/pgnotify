package pgnotify_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mickamy/pgnotify"
)

type mockExecutor struct {
	execFn func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (m *mockExecutor) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return m.execFn(ctx, sql, args...)
}

func TestNotify(t *testing.T) {
	t.Parallel()

	type event struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	tests := []struct {
		name     string
		channel  string
		payload  any
		execFn   func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
		wantSQL  string
		wantArgs []any
		wantErr  bool
	}{
		{
			name:     "sends json payload",
			channel:  "user_created",
			payload:  event{ID: 42, Name: "alice"},
			wantSQL:  "SELECT pg_notify($1, $2)",
			wantArgs: []any{"user_created", `{"id":42,"name":"alice"}`},
		},
		{
			name:    "exec error is wrapped",
			channel: "ch",
			payload: event{ID: 1, Name: "bob"},
			execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				return pgconn.CommandTag{}, errors.New("conn lost")
			},
			wantErr: true,
		},
		{
			name:    "unmarshalable payload returns error",
			channel: "ch",
			payload: func() {}, // functions cannot be marshaled to JSON
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			var gotSQL string
			var gotArgs []any

			execFn := tt.execFn
			if execFn == nil {
				execFn = func(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					gotSQL = sql
					gotArgs = args
					return pgconn.CommandTag{}, nil
				}
			}

			exec := &mockExecutor{execFn: execFn}

			err := pgnotify.Notify(ctx, exec, tt.channel, tt.payload)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("args length = %d, want %d", len(gotArgs), len(tt.wantArgs))
			}

			for i, want := range tt.wantArgs {
				got, ok := gotArgs[i].(string)
				wantStr, wantOk := want.(string)
				if !ok || !wantOk {
					t.Errorf("args[%d]: unexpected type", i)
					continue
				}
				if got != wantStr {
					t.Errorf("args[%d] = %q, want %q", i, got, wantStr)
				}
			}
		})
	}
}
