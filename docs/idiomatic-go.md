# Idiomatic Go in Practice

“How to think in Go” with concrete patterns and code you can lift into real projects.

---

## 1) What “idiomatic Go” means

Patterns you see in healthy Go code:

- Simple, explicit control flow over clever abstraction.
- Small interfaces; prefer concrete types.
- Errors as values, handled near the source.
- Composition over inheritance.
- Tool-enforced style (`gofmt`, `go vet`, linters).
- Concurrency with goroutines + channels + context.

Read and internalize *Effective Go* and *Go Code Review Comments* (the baseline for idioms).

---

## 2) Tooling & workflow

- Format/imports: always `goimports` ( = `gofmt` + import fixes). Configure your editor to run on save.
- Static checks: `go vet ./...` and `golangci-lint run ./...`.
- Modules: use `go mod init`, commit `go.mod` and `go.sum`; no GOPATH gymnastics.

---

## 3) Project layout (small/medium service)

```
myapp/
  go.mod
  cmd/myapp/
    main.go           # tiny wiring only
  internal/
    http/
      server.go
    service/
      user.go
    storage/
      postgres.go
```

- `cmd/myapp`: entrypoint, minimal wiring.
- `internal`: implementation details (compiler-enforced visibility).

`cmd/myapp/main.go`:

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"example.com/myapp/internal/http"
	"example.com/myapp/internal/service"
	"example.com/myapp/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()

	db, err := storage.OpenPostgres(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	userSvc := service.NewUserService(db, logger)
	srv := http.NewServer(userSvc, logger)

	if err := srv.Run(ctx, ":8080"); err != nil {
		logger.Error("server exited", "err", err)
		os.Exit(1)
	}
}
```

Notice how `main` only wires dependencies.

---

## 4) Naming & basic style

- Package names: short, lower-case, no underscores (`auth`, `user`, `storage`).
- Exported: `CamelCase`; unexported: `lowerCamel`.
- Keep locals short when scope is tiny (`i`, `u` in a small loop).

---

## 5) Zero values & construction

Let zero values work:

```go
type Config struct {
	Timeout time.Duration
}

func (c *Config) timeout() time.Duration {
	if c.Timeout == 0 {
		return 5 * time.Second
	}
	return c.Timeout
}
```

Use functional options only when config is non-trivial.

---

## 6) Value vs pointer receivers

- Pointer: mutates receiver or holds mutex/large data.
- Value: small, immutable helpers.

```go
type User struct {
	ID   int64
	Name string
}

func (u User) DisplayName() string { /* ... */ }

type Counter struct {
	mu sync.Mutex
	n  int
}

func (c *Counter) Inc() { /* ... */ }
```

---

## 7) Interfaces: “accept interfaces, return structs”

Define small interfaces at the consumer; constructors return concrete types.

```go
// storage/postgres.go
type PostgresStorage struct{ db *sql.DB }
func NewPostgresStorage(db *sql.DB) *PostgresStorage { return &PostgresStorage{db: db} }
func (s *PostgresStorage) GetUser(ctx context.Context, id int64) (User, error) { /* ... */ }

// service/user.go
type UserStore interface {
	GetUser(ctx context.Context, id int64) (User, error)
}

type Service struct {
	store  UserStore
	logger *slog.Logger
}
```

Keep interfaces tiny (1–3 methods).

---

## 8) Error handling

Errors are values:

```go
u, err := repo.GetUser(ctx, id)
if err != nil {
	return nil, fmt.Errorf("get user %d: %w", id, err)
}
```

Wrap with `%w`; handle at boundaries; prefer `errors.Is`/`errors.As`. Panics only for programmer bugs in `main`.

---

## 9) Context

- First parameter after receiver.
- Pass down; don’t store in structs.
- Add timeouts at the edge (HTTP handler/CLI).

```go
ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
defer cancel()
```

---

## 10) Concurrency

- Goroutines are cheap; tie them to a `context`.
- `errgroup` for related tasks with fail-fast semantics.
- Channels for pipelines/backpressure; mutexes for shared state.
- Run the race detector often: `go test -race ./...`.

```go
var g errgroup.Group
g.SetLimit(8)
for _, job := range jobs {
	job := job
	g.Go(func() error { return process(ctx, job) })
}
if err := g.Wait(); err != nil { return err }
```

---

## 11) Generics (use, don’t abuse)

Great for reusable helpers/collections; avoid over-templating business logic.

```go
func Map[T any, R any](in []T, fn func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}
```

Keep constraints small and meaningful.

---

## 12) Logging with `slog`

Structured, leveled logging in stdlib (Go 1.21+):

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))
slog.SetDefault(logger)
slog.Info("order created", "order_id", 42, "user_id", 123)
```

Log at boundaries, pass `*slog.Logger` into services.

---

## 13) Testing (table-driven & fuzz)

```go
func TestNormalizeEmail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "USER@EXAMPLE.COM", "user@example.com"},
		{"spaces", " user@example.com ", "user@example.com"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeEmail(tt.in); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
```

Add fuzz tests for parsers/serializers.

---

## 14) A quick idiomatic makeover example

Before: no context, provider-defined fat interface, unstructured logs, no error wrapping.

After (sketch):

```go
type UserStore interface {
	GetUser(ctx context.Context, id int64) (User, error)
}

type UserService struct {
	store  UserStore
	logger *slog.Logger
}

func (s *UserService) GetUserName(ctx context.Context, id int64) (string, error) {
	s.logger.Info("get user", "user_id", id)

	u, err := s.store.GetUser(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get user %d: %w", id, err)
	}
	if u.RegistrationDate.After(time.Now()) {
		return "", fmt.Errorf("user %d: %w", id, ErrInvalidRegistrationDate)
	}
	return u.Name, nil
}
```

Context-aware, structured logs, wrapped errors, tiny consumer-defined interface.

---

## 15) Idiomatic Go checklist

- `gofmt`/`goimports` applied?
- Package names short/lowercase?
- `context.Context` passed for operations that can block?
- Errors wrapped with `%w` and checked via `errors.Is/As`?
- Interfaces small and defined at the consumer?
- Goroutines tied to context/errgroup; no leaks?
- Structured logging (`slog` or similar) at boundaries?
- Generics used for helpers, not everywhere?
- Tests table-driven; run with `-race`; linters clean?

If “yes” to most, you’re in idiomatic territory.

---

Further reading:

- Effective Go — https://go.dev/doc/effective_go
- Go Code Review Comments — https://go.dev/wiki/CodeReviewComments
- Organizing a Go module — https://go.dev/doc/modules/layout
- slog intro — https://go.dev/blog/slog
- errgroup — https://pkg.go.dev/golang.org/x/sync/errgroup
