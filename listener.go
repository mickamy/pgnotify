package pgnotify

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgxlisten"
)

const defaultReconnectDelay = 5 * time.Second

// handler is a function invoked when a notification arrives on a subscribed channel.
type handler func(notification *pgconn.Notification)

// Listener wraps pgxlisten.Listener to provide channel-based subscriptions.
type Listener struct {
	connect        func(ctx context.Context) (*pgx.Conn, error)
	reconnectDelay time.Duration
	logError       func(context.Context, error)

	mu       sync.Mutex
	handlers map[string][]handler
}

// Option configures a Listener.
type Option func(*Listener)

// WithReconnectDelay sets the delay between reconnection attempts.
func WithReconnectDelay(d time.Duration) Option {
	return func(l *Listener) {
		l.reconnectDelay = d
	}
}

// WithLogError sets a callback invoked when a background error occurs.
func WithLogError(f func(context.Context, error)) Option {
	return func(l *Listener) {
		l.logError = f
	}
}

// NewListener creates a new Listener.
// The connect function is called to establish (and re-establish) the PostgreSQL connection.
func NewListener(connect func(ctx context.Context) (*pgx.Conn, error), opts ...Option) *Listener {
	l := &Listener{
		connect:        connect,
		reconnectDelay: defaultReconnectDelay,
		handlers:       make(map[string][]handler),
	}
	for _, o := range opts {
		o(l)
	}
	return l
}

// Listen starts listening for notifications and blocks until ctx is cancelled
// or an unrecoverable error occurs. All subscription channels are closed when Listen returns.
func (l *Listener) Listen(ctx context.Context) error {
	inner := &pgxlisten.Listener{
		Connect: l.connect,
	}

	if l.reconnectDelay > 0 {
		inner.ReconnectDelay = l.reconnectDelay
	}

	if l.logError != nil {
		inner.LogError = l.logError
	}

	l.mu.Lock()
	channels := make([]string, 0, len(l.handlers))
	for ch := range l.handlers {
		channels = append(channels, ch)
	}
	l.mu.Unlock()

	for _, ch := range channels {
		inner.Handle(ch, pgxlisten.HandlerFunc(func(_ context.Context, notification *pgconn.Notification, _ *pgx.Conn) error {
			l.mu.Lock()
			hs := l.handlers[ch]
			l.mu.Unlock()

			for _, h := range hs {
				h(notification)
			}
			return nil
		}))
	}

	if err := inner.Listen(ctx); err != nil {
		return fmt.Errorf("pgnotify: listen: %w", err)
	}

	return nil
}

// addHandler registers a handler for the given channel.
// It is safe to call before or during Listen.
func (l *Listener) addHandler(channel string, h handler) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.handlers[channel] = append(l.handlers[channel], h)
}
