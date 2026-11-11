// Package database manages the PostgreSQL connection pool.
//
// In Node.js (src/config/db.js):
//   const pool = new Pool({ connectionString });
//   const adapter = new PrismaPg(pool);
//   const prisma = new PrismaClient({ adapter });
//   module.exports = prisma;
//
// In Go, we use pgx/v5's pgxpool — a goroutine-safe connection pool.
// No ORM, no code generation — raw SQL through a typed driver.
//
// Key Go concepts demonstrated:
//   - Pointers (*pgxpool.Pool)
//   - context.Context for cancellable DB operations
//   - defer for cleanup
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates and validates a PostgreSQL connection pool.
//
// Concept: Pointers
//   In Go, variables hold values by default. If you pass a struct to a
//   function, Go copies it. Pointers hold the MEMORY ADDRESS of a value.
//
//   pool := pgxpool.Pool{...}    // the Pool value itself (gets copied on pass)
//   pool := &pgxpool.Pool{...}   // a pointer TO a Pool (cheaply passed around)
//
//   The & operator takes the address of a value.
//   The * operator dereferences a pointer (reads the value at the address).
//
//   Why pgxpool.Pool is a pointer:
//   - The pool manages internal goroutines and state (connection list, locks)
//   - You never want to copy this — you want ONE pool shared everywhere
//   - A pointer ensures every caller references the SAME pool instance
//
// Concept: context.Context
//   Every I/O operation in Go takes a context. The context carries:
//   1. A deadline/timeout  → "this operation must finish within 10 seconds"
//   2. A cancellation signal → "stop if the HTTP request is cancelled"
//   3. Request-scoped values → "who is the current user?"
//
//   context.Background() is the root context — it never cancels.
//   context.WithTimeout() creates a child context with a time limit.
func NewPool(databaseURL string) (*pgxpool.Pool, error) {
	// context.Background() = the top-level context (never cancels).
	// We use it here because connection setup isn't request-scoped.
	ctx := context.Background()

	// Parse the connection string into a pgx config struct.
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	// Configure the pool — these mirror what your Node.js pg.Pool does implicitly.
	config.MaxConns = 25                     // max simultaneous connections
	config.MinConns = 5                      // keep at least 5 warm connections
	config.MaxConnLifetime = 1 * time.Hour   // recycle connections hourly (avoid stale)
	config.MaxConnIdleTime = 30 * time.Minute // close idle connections after 30 min

	// Create the pool. This does NOT open connections yet.
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Ping verifies the pool can actually reach the database.
	// We give it 10 seconds — if the DB is unreachable, fail fast at startup.
	//
	// Concept: context.WithTimeout
	//   Creates a child context that auto-cancels after the duration.
	//   ALWAYS call the cancel() function when done — it releases resources.
	//   This is the Go pattern for preventing goroutine leaks from contexts.
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel() // "defer" runs this when the surrounding function returns

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close() // close the pool if ping fails to avoid orphaned connections
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
}

// Concept: defer
//   defer schedules a function call to run when the SURROUNDING FUNCTION returns,
//   no matter how it returns (normal return, error return, panic).
//
//   This is the Go equivalent of try/finally in JavaScript:
//
//   JavaScript:
//     try {
//       const resource = acquire();
//       doWork(resource);
//     } finally {
//       resource.release(); // always runs
//     }
//
//   Go:
//     resource := acquire()
//     defer resource.Release() // registered immediately, runs at function exit
//     doWork(resource)
//
//   Defers execute in LIFO order (last registered runs first).
//   This makes cleanup code sit right next to acquisition code, which is
//   much easier to audit than try/finally blocks.

