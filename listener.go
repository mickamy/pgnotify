package pgnotify

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
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
	closers  []func()
	started  atomic.Bool
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
	if connect == nil {
		panic("connect function cannot be nil")
	}
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

// ErrAlreadyListening is returned when Listen is called more than once.
var ErrAlreadyListening = errors.New("pgnotify: listener already started")

// Listen starts listening for notifications and blocks until ctx is cancelled
// or an unrecoverable error occurs. All subscription channels are closed when Listen returns.
// Listen can only be called once per Listener.
func (l *Listener) Listen(ctx context.Context) error {
	if !l.started.CompareAndSwap(false, true) {
		return ErrAlreadyListening
	}

	defer l.closeAll()

	inner := &pgxlisten.Listener{
		Connect: l.connect,
	}

	inner.ReconnectDelay = l.reconnectDelay

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
// It must be called before Listen.
func (l *Listener) addHandler(channel string, h handler) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.handlers[channel] = append(l.handlers[channel], h)
}

// addCloser registers a function to be called when Listen returns.
func (l *Listener) addCloser(f func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.closers = append(l.closers, f)
}

func (l *Listener) closeAll() {
	l.mu.Lock()
	closers := l.closers
	l.closers = nil
	l.mu.Unlock()

	for _, f := range closers {
		f()
	}
}
