// Auth middleware: JWT verification and user identity propagation.
//
// In Node.js (src/middleware/auth.middleware.js):
//   const protect = async (req, res, next) => {
//     const token = req.headers.authorization.split(' ')[1];
//     const decoded = jwt.verify(token, config.jwt.secret);
//     req.user = await prisma.user.findUnique({ where: { id: decoded.id } });
//     next();
//   }
//
// In Go, the same logic is implemented as a Gin middleware that stores
// the authenticated userID in the request context using a typed key.
//
// Key improvement over Node.js:
//   We store ONLY the userId in context (not the full user object).
//   Each handler/service fetches the user if and when it actually needs it.
//   This avoids the extra DB round-trip on every protected request.
package middleware

import (
	"context"
	"strings"

	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a custom unexported type used as context keys.
//
// Concept: Why Use a Custom Type for Context Keys?
//   context.WithValue(ctx, key, value) stores key-value pairs in a context.
//   The key can be ANY comparable type. The danger: if you use a plain string
//   like "userID", ANY package that uses the same string key can read OR
//   overwrite your value. This is a silent bug.
//
//   By defining our own private type (contextKey), only THIS package can
//   create keys of that type. No other package can accidentally collide with
//   our keys — the type system prevents it. This is called "type-safe context keys".
//
//   Node.js equivalent:
//     req.user = { id: "..." }  ← anyone can overwrite req.user
//   Go context:
//     ctx = context.WithValue(ctx, contextKeyUserID, id)  ← typed key, safe
type contextKey int

const (
	// contextKeyUserID is the key for the authenticated user's UUID.
	// Using iota makes this an auto-incrementing integer constant.
	// Since contextKey is a private type, this key can never be used
	// accidentally outside this package.
	contextKeyUserID contextKey = iota
)

// Claims represents the data we store inside a JWT token.
//
// Concept: Struct Embedding
//   jwt.RegisteredClaims is a standard struct provided by the JWT library.
//   By embedding it inside our Claims struct (without a field name),
//   we inherit all its fields (Issuer, Subject, ExpiresAt, etc.)
//   as if they were fields of our Claims struct.
//
//   This is Go's composition model — not inheritance, but embedding.
//   It's similar to JavaScript's object spread:
//     const claims = { ...standardClaims, userID: "abc" }
//   But typed and checked at compile time.
type Claims struct {
	UserID string `json:"id"` // matches Node.js: jwt.sign({ id }, secret)
	jwt.RegisteredClaims
}

// Protect is the authentication middleware.
// Every route that calls router.Use(middleware.Protect(secret)) or
// uses it on individual routes is protected.
func Protect(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- Step 1: Extract the token from the Authorization header ---
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Error(c, 401, "Not authorized, no token")
			c.Abort() // stop the chain — do NOT call c.Next()
			return
		}

		// strings.CutPrefix is cleaner than .split(' ')[1]
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// --- Step 2: Verify and parse the JWT ---
		// jwt.ParseWithClaims verifies the signature AND populates our Claims struct.
		// The function argument is the "key function" — it returns the signing key.
		// This design supports multiple keys (e.g., key rotation) cleanly.
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			// Verify the signing method is what we expect (HMAC-SHA256).
			// Never skip this check — an attacker could forge tokens with "alg: none".
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apperror.Unauthorized("unexpected signing method")
			}
			// Return the secret key as a byte slice.
			// []byte(jwtSecret) converts the string to []byte — jwt library expects bytes.
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			// Check specific error types for better error messages.
			if err != nil && strings.Contains(err.Error(), "token is expired") {
				response.Error(c, 401, "Not authorized, token expired")
			} else {
				response.Error(c, 401, "Not authorized, token failed")
			}
			c.Abort()
			return
		}

		// --- Step 3: Store userId in context ---
		// We use c.Request.Context() to get the request's context, then
		// create a new context with the userID stored under our typed key.
		//
		// Why not just c.Set("userID", claims.UserID)?
		// c.Set() stores in Gin's own map, which is fine. But storing in
		// context.Context means services and repositories (which only receive
		// context, not *gin.Context) can still read the user ID without
		// depending on Gin at all. This keeps our business logic Gin-free.
		ctx := context.WithValue(c.Request.Context(), contextKeyUserID, claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		c.Next() // authenticated — proceed to the actual handler
	}
}

// GetUserID extracts the authenticated user's ID from a context.
// Returns empty string if not found (which should never happen on protected routes).
//
// Usage in handlers:
//   userID := middleware.GetUserID(c.Request.Context())
//
// Usage in services (no Gin dependency!):
//   userID := middleware.GetUserID(ctx)
func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(contextKeyUserID).(string)
	return id
}

