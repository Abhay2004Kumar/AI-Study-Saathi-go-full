// Package service contains the business logic layer.
//
// In Node.js, services are plain modules with async functions.
// In Go, we define interfaces for each service, then implement them as structs.
//
// Why interfaces?
//   An interface in Go defines a CONTRACT — a set of method signatures
//   that any type must implement to be "considered" that interface.
//
//   type AuthService interface {
//       Register(ctx, req) (*dto.AuthResponse, error)
//       Login(ctx, req) (*dto.AuthResponse, error)
//       GetMe(ctx, userID) (*dto.UserResponse, error)
//   }
//
//   Benefits:
//   1. Testability: inject a mock, test the handler without hitting the DB
//   2. Flexibility: swap implementations (e.g., add caching) without changing handlers
//   3. Documentation: the interface IS the contract, readable at a glance
//   4. Decoupling: handler package imports the interface, not the implementation
package service

import (
	"context"

	"github.com/aiStudySaathi/backend/internal/dto"
)

// AuthService defines all authentication-related operations.
// Handlers depend on this interface, not on the concrete authService struct.
//
// Concept: Implicit Interface Implementation
//   In Java/C#, you explicitly declare "implements AuthService".
//   In Go, there is NO such declaration. If a type has all the methods
//   defined in the interface, it automatically satisfies it.
//
//   This is called "structural typing" or "duck typing" but enforced at compile time.
//   If your authService struct is missing a method, the compiler tells you exactly
//   which method is missing when you try to assign it to the interface variable.
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	GetMe(ctx context.Context, userID string) (*dto.UserResponse, error)
}

