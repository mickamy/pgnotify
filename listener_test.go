package pgnotify_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mickamy/pgnotify"
)

func TestListener_addHandler(t *testing.T) {
	t.Parallel()

	l := pgnotify.NewListener(pgnotify.NopConnect)

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

func TestNewListener_panics_on_nil_connect(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil connect")
		}
	}()

	pgnotify.NewListener(nil)
}

func TestNewListener_options(t *testing.T) {
	t.Parallel()

	t.Run("WithReconnectDelay sets delay", func(t *testing.T) {
		t.Parallel()

		l := pgnotify.NewListener(pgnotify.NopConnect, pgnotify.WithReconnectDelay(3*time.Second))

		if got := l.ExportReconnectDelay(); got != 3*time.Second {
			t.Errorf("reconnectDelay = %v, want 3s", got)
		}
	})

	t.Run("WithLogError sets callback", func(t *testing.T) {
		t.Parallel()

		l := pgnotify.NewListener(pgnotify.NopConnect, pgnotify.WithLogError(func(_ context.Context, _ error) {}))

		if !l.ExportHasLogError() {
			t.Error("expected logError to be set")
		}
	})

	t.Run("defaults without options", func(t *testing.T) {
		t.Parallel()

		l := pgnotify.NewListener(pgnotify.NopConnect)

		if l.ExportHasLogError() {
			t.Error("expected logError to be nil by default")
		}
	})
}
