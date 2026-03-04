package pgnotify_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mickamy/pgnotify"
)

func TestListener_addHandler(t *testing.T) {
	t.Parallel()

	l := pgnotify.NewListener(nil)

	var called atomic.Int32

	h := func(_ *pgconn.Notification) {
		called.Add(1)
	}

	l.ExportAddHandler("test_channel", h)
	l.ExportAddHandler("test_channel", h)

	handlers := l.ExportHandlers("test_channel")
	if len(handlers) != 2 {
		t.Fatalf("handlers count = %d, want 2", len(handlers))
	}

	for _, fn := range handlers {
		fn(&pgconn.Notification{Channel: "test_channel", Payload: `{}`})
	}

	if got := called.Load(); got != 2 {
		t.Errorf("called = %d, want 2", got)
	}
}

func TestNewListener_options(t *testing.T) {
	t.Parallel()

	var logCalled bool

	l := pgnotify.NewListener(
		nil,
		pgnotify.WithReconnectDelay(0),
		pgnotify.WithLogError(func(_ context.Context, _ error) {
			logCalled = true
		}),
	)

	if l == nil {
		t.Fatal("NewListener returned nil")
	}

	_ = logCalled
}
