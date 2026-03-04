package pgnotify

import "github.com/jackc/pgx/v5/pgconn"

// ExportAddHandler exposes addHandler for testing.
func (l *Listener) ExportAddHandler(channel string, h func(*pgconn.Notification)) {
	l.addHandler(channel, h)
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
