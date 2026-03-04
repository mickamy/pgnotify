package pgnotify_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mickamy/pgnotify"
)

func TestSubscribe(t *testing.T) {
	t.Parallel()

	type event struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	t.Run("delivers decoded event to channel", func(t *testing.T) {
		t.Parallel()

		l := pgnotify.NewListener(nil)
		ch := pgnotify.Subscribe[event](l, "user_created")

		handlers := l.ExportHandlers("user_created")
		if len(handlers) != 1 {
			t.Fatalf("handlers count = %d, want 1", len(handlers))
		}

		handlers[0](&pgconn.Notification{
			Channel: "user_created",
			Payload: `{"id":42,"name":"alice"}`,
		})

		got := <-ch
		if got.ID != 42 || got.Name != "alice" {
			t.Errorf("got %+v, want {ID:42 Name:alice}", got)
		}
	})

	t.Run("json decode error calls error handler", func(t *testing.T) {
		t.Parallel()

		var gotErr error
		l := pgnotify.NewListener(nil)
		_ = pgnotify.Subscribe[event](l, "ch", pgnotify.WithErrorHandler(func(err error) {
			gotErr = err
		}))

		handlers := l.ExportHandlers("ch")
		handlers[0](&pgconn.Notification{
			Channel: "ch",
			Payload: `not json`,
		})

		if gotErr == nil {
			t.Fatal("expected error handler to be called")
		}
	})

	t.Run("buffer full drops message and calls error handler", func(t *testing.T) {
		t.Parallel()

		var gotErr error
		var mu sync.Mutex
		l := pgnotify.NewListener(nil)
		_ = pgnotify.Subscribe[event](l, "ch",
			pgnotify.WithBufferSize(1),
			pgnotify.WithErrorHandler(func(err error) {
				mu.Lock()
				gotErr = err
				mu.Unlock()
			}),
		)

		handlers := l.ExportHandlers("ch")

		// Fill the buffer.
		handlers[0](&pgconn.Notification{
			Channel: "ch",
			Payload: `{"id":1,"name":"a"}`,
		})

		// This should be dropped.
		handlers[0](&pgconn.Notification{
			Channel: "ch",
			Payload: `{"id":2,"name":"b"}`,
		})

		mu.Lock()
		defer mu.Unlock()

		var bufErr *pgnotify.BufferFullError
		if !errors.As(gotErr, &bufErr) {
			t.Fatalf("expected BufferFullError, got %v", gotErr)
		}

		if bufErr.Channel != "ch" {
			t.Errorf("channel = %q, want %q", bufErr.Channel, "ch")
		}
	})

	t.Run("channel is closed after closeAll", func(t *testing.T) {
		t.Parallel()

		l := pgnotify.NewListener(nil)
		ch := pgnotify.Subscribe[event](l, "ch")

		l.ExportCloseAll()

		_, ok := <-ch
		if ok {
			t.Fatal("expected channel to be closed")
		}
	})

	t.Run("no error handler does not panic", func(t *testing.T) {
		t.Parallel()

		l := pgnotify.NewListener(nil)
		_ = pgnotify.Subscribe[event](l, "ch")

		handlers := l.ExportHandlers("ch")

		// Invalid JSON without error handler should not panic.
		handlers[0](&pgconn.Notification{
			Channel: "ch",
			Payload: `bad`,
		})
	})
}
