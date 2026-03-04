package pgnotify

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgconn"
)

const defaultBufferSize = 64

type subscribeConfig struct {
	bufferSize   int
	errorHandler func(error)
}

// SubscribeOption configures a subscription.
type SubscribeOption func(*subscribeConfig)

// WithBufferSize sets the channel buffer size. Default is 64.
func WithBufferSize(n int) SubscribeOption {
	return func(c *subscribeConfig) {
		c.bufferSize = n
	}
}

// WithErrorHandler sets a callback invoked when JSON decoding fails or the buffer is full.
func WithErrorHandler(f func(error)) SubscribeOption {
	return func(c *subscribeConfig) {
		c.errorHandler = f
	}
}

// Subscribe registers a typed subscription on the given channel and returns a receive-only channel.
// It must be called before Listener.Listen. The returned channel is closed when Listen returns.
//
// If the internal buffer is full, the notification is dropped and the error handler is called.
func Subscribe[T any](l *Listener, channel string, opts ...SubscribeOption) <-chan T {
	cfg := subscribeConfig{
		bufferSize: defaultBufferSize,
	}
	for _, o := range opts {
		o(&cfg)
	}

	ch := make(chan T, cfg.bufferSize)
	l.addCloser(func() { close(ch) })

	l.addHandler(channel, func(notification *pgconn.Notification) {
		var v T
		if err := json.Unmarshal([]byte(notification.Payload), &v); err != nil {
			if cfg.errorHandler != nil {
				cfg.errorHandler(err)
			}
			return
		}

		select {
		case ch <- v:
		default:
			if cfg.errorHandler != nil {
				cfg.errorHandler(&BufferFullError{Channel: channel})
			}
		}
	})

	return ch
}

// BufferFullError is returned via the error handler when the subscription buffer is full.
type BufferFullError struct {
	Channel string
}

func (e *BufferFullError) Error() string {
	return "pgnotify: buffer full for channel " + e.Channel
}
