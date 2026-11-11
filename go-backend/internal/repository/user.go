// UserRepository implementation using pgx/v5.
//
// This is where we write actual SQL — no magic, no reflection,
// no code generation. Just explicit queries and row scanning.
//
// Key Go concepts demonstrated:
//   - pgxpool.Pool for connection management
//   - pgx.Row scanning into structs
//   - Handling "no rows" vs actual errors
//   - time.Time handling with PostgreSQL timestamps
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxUserRepository is the concrete implementation of UserRepository.
// It's unexported (lowercase) — callers use the UserRepository interface.
type pgxUserRepository struct {
	pool *pgxpool.Pool // the connection pool, shared across all requests
}

// NewUserRepository creates a new UserRepository backed by pgx.
// Returns the interface (not the concrete type) for clean dependency injection.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &pgxUserRepository{pool: pool}
}

// FindByEmail looks up a user by their email address.
//
// Prisma equivalent:
//   prisma.user.findUnique({ where: { email } })
//
// In Go, we write the SQL ourselves:
//   1. pool.QueryRow() runs the query and returns a single row
//   2. row.Scan() reads the columns into our variables
//   3. pgx.ErrNoRows is returned when no row matches — not an error, just "not found"
func (r *pgxUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	// QueryRow executes a query that is expected to return at most one row.
	// $1 is the placeholder for the first argument (email).
	// pgx uses $1, $2, $3... (PostgreSQL-style), not ? (MySQL-style).
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, email, "passwordHash", "createdAt", "updatedAt"
		 FROM "User"
		 WHERE email = $1`,
		email,
	)

	// scan reads the row's columns into our variables in ORDER.
	// The order must match the SELECT column list exactly.
	user := &model.User{}
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		// pgx.ErrNoRows is returned when QueryRow finds no matching rows.
		// This is NOT a database error — it's the "not found" case.
		// We return (nil, nil) to signal "user not found" without an error.
		//
		// Node.js Prisma equivalent:
		//   prisma.user.findUnique() returns null when not found — no exception
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // nil user, nil error = "not found"
		}
		// Any other error is a real database problem
		return nil, fmt.Errorf("querying user by email: %w", err)
	}

	return user, nil
}

// FindByID looks up a user by their UUID.
func (r *pgxUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, email, "passwordHash", "createdAt", "updatedAt"
		 FROM "User"
		 WHERE id = $1`,
		id,
	)

	user := &model.User{}
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying user by id: %w", err)
	}

	return user, nil
}

// Create inserts a new user into the database.
//
// Prisma equivalent:
//   prisma.user.create({ data: { name, email, passwordHash } })
//
// We use RETURNING to get the generated timestamps back from PostgreSQL.
// This is better than a separate SELECT after INSERT — one round trip.
func (r *pgxUserRepository) Create(ctx context.Context, user *model.User) error {
	now := time.Now().UTC()

	// RETURNING "createdAt", "updatedAt" reads the DB-generated timestamps
	// back into our struct immediately after the INSERT.
	// This keeps our in-memory user struct in sync with what's in the DB.
	err := r.pool.QueryRow(ctx,
		`INSERT INTO "User" (id, name, email, "passwordHash", "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING "createdAt", "updatedAt"`,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
		now,
		now,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}

	return nil
}

// Concept: Why not use pgx.NamedArgs or pgx.RowToStruct?
//   pgx v5 supports scanning directly into structs using pgx.RowToStructByName.
//   We write manual Scan() calls here for two reasons:
//   1. Learning: seeing the mapping explicitly teaches you what SQL does
//   2. Control: you decide exactly which columns to fetch and how to map them
//   In a production project, you might use pgx.RowToStructByName for brevity.

