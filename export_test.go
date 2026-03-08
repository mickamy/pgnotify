package pgnotify

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// NopConnect is a connect function for testing that always returns an error.
// It satisfies the non-nil requirement of NewListener without needing a real database.
var NopConnect = func(_ context.Context) (*pgx.Conn, error) {
	return nil, errors.New("nop connect: not implemented")
}

// ExportAddHandler exposes addHandler for testing.
func (l *Listener) ExportAddHandler(channel string, h func(*pgconn.Notification)) {
	l.addHandler(channel, h)
}

// ExportCloseAll exposes closeAll for testing.
func (l *Listener) ExportCloseAll() {
	l.closeAll()
}

// ExportReconnectDelay returns the configured reconnect delay.
func (l *Listener) ExportReconnectDelay() time.Duration {
	return l.reconnectDelay
}

// ExportHasLogError returns whether a logError callback is configured.
func (l *Listener) ExportHasLogError() bool {
	return l.logError != nil
}

// ExportHandlers returns the registered handlers for a channel.
func (l *Listener) ExportHandlers(channel string) []func(*pgconn.Notification) {
	l.mu.Lock()
	defer l.mu.Unlock()

	hs := l.handlers[channel]
	out := make([]func(*pgconn.Notification), len(hs))
	for i, h := range hs {
		out[i] = h
	}
	return out
}
