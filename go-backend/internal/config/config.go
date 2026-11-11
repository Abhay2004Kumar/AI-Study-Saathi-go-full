// Package config is responsible for loading and validating all environment
// variables the application needs. This is the Go equivalent of your
// Node.js src/config/env.js file.
//
// Key Go concepts demonstrated here:
//   - Structs as typed configuration containers
//   - Methods on structs (receiver functions)
//   - os.LookupEnv for safe environment variable reading
//   - strconv for string-to-number conversion
//   - Fail-fast validation at startup
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds ALL validated configuration for the application.
//
// In Node.js you exported a plain object:
//   module.exports = { env: envVars.NODE_ENV, port: envVars.PORT, ... }
//
// In Go, we use a struct. The difference is that struct fields have
// TYPES enforced at compile time. You cannot accidentally assign a string
// to Port — the compiler will reject it before the program even runs.
//
// Naming convention: exported fields start with uppercase (Port, not port).
// In Go, uppercase = public (accessible from other packages),
//         lowercase = private (only accessible within this package).
type Config struct {
	// Server
	Env         string // "development", "production", "test"
	Port        int    // HTTP port number
	FrontendURL string // e.g. "http://localhost:8081"

	// Database
	DatabaseURL string // Full PostgreSQL connection string

	// JWT
	JWTSecret    string // Signing key — NEVER log this
	JWTExpiresIn string // e.g. "7d", "24h"

	// Gemini AI
	GeminiAPIKey string // Google Gemini API key — NEVER log this

	// Upload
	UploadDir    string // Directory where uploaded files are stored
	MaxFileSizeMB int   // Maximum allowed file size
}

// Load reads environment variables, validates required ones, and returns
// a fully-populated Config. If any required variable is missing, it
// returns an error — the caller (main.go) will log the error and exit.
//
// In Node.js (src/config/env.js):
//   const { error, value } = envVarsSchema.validate(process.env);
//   if (error) throw new Error(`Config validation error: ${error.message}`);
//
// In Go, we return an error instead of throwing. The caller decides
// whether to log-and-exit or handle differently. This is Go's explicit
// error handling philosophy — control flow is never hidden.
func Load() (*Config, error) {
	// godotenv.Load() reads a .env file and populates os.Getenv.
	// If the file doesn't exist (production), it's not an error —
	// real env vars are set by the process/container already.
	// The underscore (_) discards the returned error intentionally here.
	_ = godotenv.Load(".env")               // try local .env first
	_ = godotenv.Load("../backend/.env")     // fallback to sibling Node.js .env

	cfg := &Config{}

	// --- Server ---
	cfg.Env = getEnvWithDefault("NODE_ENV", "development")

	portStr := getEnvWithDefault("PORT", "5000")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		// strconv.Atoi returns an error if PORT="abc".
		// fmt.Errorf creates a new error with a formatted message.
		// The %w verb "wraps" the original error so callers can inspect it.
		return nil, fmt.Errorf("invalid PORT value %q: %w", portStr, err)
	}
	cfg.Port = port
	cfg.FrontendURL = strings.TrimSpace(getEnvWithDefault("FRONTEND_URL", "http://localhost:8081"))

	// --- Database ---
	// requireEnv returns an error if the variable is empty.
	// We collect errors lazily instead of panicking on the first one,
	// so the user sees ALL missing variables at once, not just the first.
	var errs []error

	cfg.DatabaseURL, err = requireEnv("DATABASE_URL")
	if err != nil {
		errs = append(errs, err)
	}

	// --- JWT ---
	cfg.JWTSecret, err = requireEnv("JWT_SECRET")
	if err != nil {
		errs = append(errs, err)
	}
	cfg.JWTExpiresIn = getEnvWithDefault("JWT_EXPIRES_IN", "7d")

	// --- Gemini ---
	cfg.GeminiAPIKey, err = requireEnv("GEMINI_API_KEY")
	if err != nil {
		errs = append(errs, err)
	}

	// --- Upload ---
	cfg.UploadDir = getEnvWithDefault("UPLOAD_DIR", "./uploads")
	cfg.MaxFileSizeMB = 10

	// If we collected any validation errors, join them and return.
	if len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed:\n%v", errs)
	}

	return cfg, nil
}

// IsProduction returns true when the application runs in production mode.
// Using a method instead of comparing cfg.Env == "production" everywhere
// makes the intent clearer and the comparison impossible to typo.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// IsDevelopment returns true in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// --- Helpers ---

// requireEnv reads an environment variable and returns an error if it's empty.
// This is the Go equivalent of joi.string().required() in your Node.js config.
func requireEnv(key string) (string, error) {
	// os.LookupEnv returns (value, exists). This is safer than os.Getenv
	// because it distinguishes between "variable not set" and "variable set to empty string".
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}
	return val, nil
}

// getEnvWithDefault reads an env var, returning defaultVal if not set.
// This is the Go equivalent of joi.string().default("development").
func getEnvWithDefault(key, defaultVal string) string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return defaultVal
	}
	return val
}

