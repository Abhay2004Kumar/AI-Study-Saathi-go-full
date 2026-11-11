# AI Study Saathi Go Backend

## Go port of the Node.js/Express backend.

### Run

```bash
cd cmd/server
go run .
```

### Build

```bash
go build ./cmd/server/
```

### Test

```bash
go test ./...
```

### Project Structure

```
cmd/server/main.go          → Entry point, wires all dependencies, starts server
internal/config/            → Typed env var loading
internal/middleware/        → Auth, error handling, logging
internal/handler/           → HTTP request handlers (like Express controllers)
internal/service/           → Business logic (interface + implementation)
internal/repository/        → Database access (interface + implementation)
internal/model/             → Database row structs
internal/dto/               → API request/response shapes
internal/routes/            → Route registration
internal/worker/            → Background workers (document processing)
pkg/apperror/               → Custom error type with HTTP status code
pkg/response/               → Standardised JSON response helpers
pkg/gemini/                 → Gemini API HTTP client
```

