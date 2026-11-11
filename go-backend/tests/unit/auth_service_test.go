// Package service_test contains unit tests for the service layer.
//
// In Node.js, you used Jest:
//   describe('AuthService', () => {
//     it('should hash password and return token', async () => { ... });
//   });
//
// In Go, testing is built into the standard library (`testing` package).
// Test files MUST end with `_test.go`.
// Test functions MUST start with `Test` and take `t *testing.T`.
//
// To run tests:
//   go test ./...               // run all tests
//   go test -v ./...            // verbose output
//   go test -run TestRegister   // run only tests matching "TestRegister"
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository is a mock implementation of repository.UserRepository.
// In Go, you don't need a mocking library like Jest.fn() — you can just
// define a struct that implements the interface.
//
// This is the beauty of Go interfaces:
// MockUserRepository has FindByEmail, FindByID, and Create methods.
// Therefore, it SATISFIES repository.UserRepository.
// No inheritance, no special mocking library. Just plain Go code.
type MockUserRepository struct {
	usersByEmail map[string]*model.User
	usersByID    map[string]*model.User
	createErr    error
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		usersByEmail: make(map[string]*model.User),
		usersByID:    make(map[string]*model.User),
	}
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return m.usersByEmail[email], nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	return m.usersByID[id], nil
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.usersByEmail[user.Email] = user
	m.usersByID[user.ID] = user
	return nil
}

// TestRegisterSuccess tests the happy path for user registration.
func TestRegisterSuccess(t *testing.T) {
	// Arrange
	mockRepo := NewMockUserRepository()
	authSvc := service.NewAuthService(mockRepo, "testsecret123", 24*time.Hour)

	req := dto.RegisterRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
	}

	// Act
	resp, err := authSvc.Register(context.Background(), req)

	// Assert
	// In Go, assertions are just if statements with t.Errorf / t.Fatalf.
	// t.Fatalf stops the test immediately.
	// t.Errorf logs the error but continues running the test.
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response to be non-nil")
	}
	if resp.Email != req.Email {
		t.Errorf("expected email %q, got %q", req.Email, resp.Email)
	}
	if resp.Name != req.Name {
		t.Errorf("expected name %q, got %q", req.Name, resp.Name)
	}
	if resp.Token == "" {
		t.Error("expected non-empty JWT token")
	}

	// Verify user was actually stored in the repository
	saved := mockRepo.usersByEmail[req.Email]
	if saved == nil {
		t.Fatal("expected user to be saved in repository")
	}
	// Verify password was hashed (not stored in plain text!)
	if saved.PasswordHash == req.Password {
		t.Error("password was stored in plain text — must be hashed!")
	}
	// Verify the hash is valid bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(saved.PasswordHash), []byte(req.Password)); err != nil {
		t.Errorf("saved password hash does not match original password: %v", err)
	}
}

// TestRegisterDuplicateEmail tests that registering with an existing email returns 409 Conflict.
func TestRegisterDuplicateEmail(t *testing.T) {
	mockRepo := NewMockUserRepository()
	// Pre-populate with an existing user
	mockRepo.usersByEmail["existing@example.com"] = &model.User{
		ID:    "user-1",
		Email: "existing@example.com",
	}

	authSvc := service.NewAuthService(mockRepo, "testsecret123", 24*time.Hour)

	req := dto.RegisterRequest{
		Name:     "Duplicate",
		Email:    "existing@example.com",
		Password: "password123",
	}

	_, err := authSvc.Register(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}

	// Verify the error is an AppError with 409 Conflict
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.StatusCode != 409 {
		t.Errorf("expected status code 409, got %d", appErr.StatusCode)
	}
}

// TestLoginSuccess tests valid login.
func TestLoginSuccess(t *testing.T) {
	mockRepo := NewMockUserRepository()

	// Hash a known password
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), 10)
	mockRepo.usersByEmail["user@example.com"] = &model.User{
		ID:           "user-1",
		Name:         "User",
		Email:        "user@example.com",
		PasswordHash: string(hash),
	}

	authSvc := service.NewAuthService(mockRepo, "testsecret123", 24*time.Hour)

	resp, err := authSvc.Login(context.Background(), dto.LoginRequest{
		Email:    "user@example.com",
		Password: "correctpassword",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

// TestLoginWrongPassword tests invalid password.
func TestLoginWrongPassword(t *testing.T) {
	mockRepo := NewMockUserRepository()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), 10)
	mockRepo.usersByEmail["user@example.com"] = &model.User{
		Email:        "user@example.com",
		PasswordHash: string(hash),
	}

	authSvc := service.NewAuthService(mockRepo, "testsecret123", 24*time.Hour)

	_, err := authSvc.Login(context.Background(), dto.LoginRequest{
		Email:    "user@example.com",
		Password: "wrongpassword",
	})

	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 401 {
		t.Errorf("expected 401 Unauthorized, got: %v", err)
	}
}

