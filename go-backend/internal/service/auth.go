// AuthService implementation — the actual business logic.
//
// This maps directly to Node.js src/controllers/auth.controller.js logic,
// but properly separated from HTTP concerns (no req/res here).
//
// Key Go concepts demonstrated:
//   - Methods on structs (receiver functions)
//   - Multiple return values (value, error)
//   - Interfaces and struct embedding
//   - golang.org/x/crypto/bcrypt usage
//   - JWT token generation
//   - time.Duration arithmetic
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// authService is the concrete implementation of AuthService.
// It's lowercase (unexported) intentionally — external packages work with
// the AuthService interface, not this concrete type. This enforces the
// interface contract from the caller's perspective.
type authService struct {
	userRepo  repository.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

// NewAuthService creates a new authService with its dependencies.
// Returns the AuthService interface (not *authService), which enforces
// that callers only use the interface methods.
//
// The Go compiler will error here if authService doesn't implement
// all the methods in AuthService. This is your compile-time safety net.
func NewAuthService(userRepo repository.UserRepository, jwtSecret string, jwtExpiry time.Duration) AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

// Register creates a new user account.
//
// Node.js equivalent (from auth.controller.js):
//   const userExists = await prisma.user.findUnique({ where: { email } });
//   if (userExists) { res.status(400); throw new Error('User already exists'); }
//   const passwordHash = await bcrypt.hash(password, 10);
//   const user = await prisma.user.create({ data: { name, email, passwordHash } });
//   res.status(201).json({ ...user, token: generateToken(user.id) });
//
// In Go:
//   - No try/catch — errors are return values, checked explicitly
//   - No async/await — Go goroutines handle concurrency differently
//   - Each error check is a deliberate decision point
func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	// --- Step 1: Check if user already exists ---
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Unexpected DB error — wrap it with context for debugging
		return nil, fmt.Errorf("checking existing user: %w", err)
	}
	if existing != nil {
		// User exists — this is a business logic error (not a DB error)
		// AppError carries the HTTP status code with it
		return nil, apperror.Conflict("User already exists")
	}

	// --- Step 2: Hash the password ---
	//
	// bcrypt.GenerateFromPassword is Go's equivalent of:
	//   await bcrypt.genSalt(10); await bcrypt.hash(password, salt)
	//
	// Cost 12 is slightly higher than 10 — more secure, slightly slower.
	// bcrypt's cost factor doubles compute time per increment:
	//   cost 10 = baseline, cost 12 = 4x slower per hash
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	// --- Step 3: Create the user ---
	user := &model.User{
		ID:           uuid.New().String(), // generate a new UUID
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashBytes),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// --- Step 4: Generate JWT and return ---
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &dto.AuthResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Token: token,
	}, nil
}

// Login authenticates a user and returns a JWT.
func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	// --- Step 1: Find the user ---
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}
	if user == nil {
		// Don't reveal whether the email exists — always say "invalid credentials"
		return nil, apperror.Unauthorized("Invalid email or password")
	}

	// --- Step 2: Compare password ---
	//
	// bcrypt.CompareHashAndPassword is Go's equivalent of:
	//   await bcrypt.compare(password, user.passwordHash)
	//
	// It returns nil on match, an error on mismatch.
	// The error is NOT a system error — it's the expected "wrong password" case.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperror.Unauthorized("Invalid email or password")
	}

	// --- Step 3: Generate JWT ---
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &dto.AuthResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Token: token,
	}, nil
}

// GetMe fetches the current user's profile.
func (s *authService) GetMe(ctx context.Context, userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}
	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// generateToken creates a signed JWT for a user.
//
// Node.js equivalent (from auth.controller.js):
//   jwt.sign({ id }, config.jwt.secret, { expiresIn: config.jwt.expiresIn })
//
// In Go, the Claims struct is explicit and type-safe.
// The expiry is a time.Duration, not a string like "7d" — no parsing needed.
func (s *authService) generateToken(userID string) (string, error) {
	now := time.Now()

	// jwt.Claims is a struct — we fill it with our data.
	// RegisteredClaims handles standard JWT fields (Issuer, ExpiresAt, etc.)
	claims := &middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			// time.Now().Add(s.jwtExpiry) is Go's way of saying "now + 7 days"
			// time.Duration arithmetic: 7 * 24 * time.Hour = 7 days
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	// jwt.NewWithClaims creates an unsigned token.
	// token.SignedString signs it with our secret key.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}

	return signed, nil
}

