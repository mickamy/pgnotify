# pgnotify

Type-safe, channel-based Go wrapper for PostgreSQL LISTEN/NOTIFY powered by [pgxlisten](https://github.com/jackc/pgxlisten) and generics.

## Features

- **Type-safe subscriptions** via Go generics — JSON payloads are automatically decoded into your struct
- **Channel-based API** — receive notifications through Go channels
- **Non-blocking delivery** — full buffers drop messages instead of blocking the listener
- **Automatic reconnection** — powered by pgxlisten
- **Works with any pgx executor** — `*pgx.Conn`, `*pgxpool.Pool`, `pgx.Tx`

## Install

```bash
go get github.com/mickamy/pgnotify
```

Requires Go 1.25+.

## Usage

### Subscribe to notifications

```go
listener := pgnotify.NewListener(func(ctx context.Context) (*pgx.Conn, error) {
    return pgx.Connect(ctx, dsn)
})

type UserCreated struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// Subscribe must be called before Listen.
ch := pgnotify.Subscribe[UserCreated](listener, "user_created")

go func() {
    if err := listener.Listen(ctx); err != nil {
        log.Fatal(err)
    }
}()

for event := range ch {
    fmt.Printf("User created: %+v\n", event)
}
```

### Send notifications

```go
type UserCreated struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// Works with *pgx.Conn, *pgxpool.Pool, pgx.Tx, etc.
err := pgnotify.Notify(ctx, pool, "user_created", UserCreated{
    ID:   42,
    Name: "alice",
})
```

This executes:

```sql
SELECT pg_notify('user_created', '{"id":42,"name":"alice"}')
```

## Options

### Listener options

```go
listener := pgnotify.NewListener(connect,
    pgnotify.WithReconnectDelay(10 * time.Second), // default: 5s
    pgnotify.WithLogError(func(ctx context.Context, err error) {
        slog.ErrorContext(ctx, "pgnotify", "error", err)
    }),
)
```

### Subscribe options

```go
ch := pgnotify.Subscribe[Event](listener, "events",
    pgnotify.WithBufferSize(128),              // default: 64
    pgnotify.WithErrorHandler(func(err error) {
        log.Println("subscribe error:", err)   // JSON decode failure or buffer full
    }),
)
```

## Design

- **Non-blocking send**: When the channel buffer is full, the notification is dropped and the error handler receives a `*BufferFullError`. This prevents a slow consumer from blocking all notifications.
- **Channel close**: All subscription channels are closed when `Listen` returns, so `for range ch` loops terminate cleanly.
- **Subscribe before Listen**: All `Subscribe` calls must happen before `Listen` is called. Subscriptions registered after `Listen` starts are not delivered.

## License

[MIT](./LICENSE)
