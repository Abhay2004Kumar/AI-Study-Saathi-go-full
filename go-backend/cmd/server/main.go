// main.go — the application entry point.
//
// In Node.js (src/server.js):
//   const app = require('./app');
//   app.listen(config.port, () => console.log('Server running'));
//   process.on('unhandledRejection', (err) => server.close(() => process.exit(1)));
//
// In Go, main() is the entry point for the entire program.
// Everything flows from here: config → database → dependencies → server.
//
// Key improvements over Node.js:
//   1. GRACEFUL SHUTDOWN: We wait for in-flight requests to finish
//      before exiting. Node.js requires complex Promise orchestration for this.
//   2. FAIL-FAST STARTUP: If anything fails to initialize, we log and exit
//      immediately with a clear error message.
//   3. EXPLICIT WIRING: Every dependency is explicitly constructed and injected.
//      No hidden globals, no require() side effects.
//
// Key Go concepts demonstrated:
//   - os.Signal and channel-based signal handling
//   - context.WithTimeout for graceful shutdown deadline
//   - http.Server for fine-grained server control
//   - Goroutines for running the server without blocking main()
//   - sync.WaitGroup (conceptually, via server.ListenAndServe goroutine)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aiStudySaathi/backend/internal/config"
	"github.com/aiStudySaathi/backend/internal/database"
	"github.com/aiStudySaathi/backend/internal/handler"
	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/routes"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/internal/service/ai"
	"github.com/aiStudySaathi/backend/internal/worker"
	"github.com/aiStudySaathi/backend/pkg/gemini"
	"github.com/gin-gonic/gin"
)

func main() {
	// =========================================================================
	// STEP 1: Setup structured logging
	// =========================================================================
	//
	// log/slog is the standard structured logging package (Go 1.21+).
	// slog.Info(), slog.Error(), slog.Warn() write key-value log entries.
	//
	// This replaces console.log() in Node.js with structured, queryable logs.
	// In production, you'd switch to slog.NewJSONHandler for JSON output.
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger) // set as the global default logger

	slog.Info("Starting AI Study Saathi Go backend")

	// =========================================================================
	// STEP 2: Load and validate configuration
	// =========================================================================
	//
	// config.Load() reads all env vars and returns a typed struct.
	// If any required variable is missing, we exit immediately.
	//
	// This is the Go equivalent of:
	//   const { error, value } = envVarsSchema.validate(process.env);
	//   if (error) throw new Error(error.message);
	cfg, err := config.Load()
	if err != nil {
		// slog.Error writes to stderr with structured fields.
		// os.Exit(1) terminates the process — no cleanup, intentional here
		// because we failed before ANY resources were acquired.
		slog.Error("Configuration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Configuration loaded", "env", cfg.Env, "port", cfg.Port)

	// =========================================================================
	// STEP 3: Connect to the database
	// =========================================================================
	//
	// database.NewPool() opens a connection pool to PostgreSQL.
	// If this fails, there's no point running the server.
	pool, err := database.NewPool(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	// defer pool.Close() ensures the pool is closed when main() returns.
	// This releases all database connections cleanly on shutdown.
	defer pool.Close()
	slog.Info("Database connected")

	// Auto-migrate database schema (creates tables, extensions, indexes if they don't exist)
	migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer migrateCancel()
	if err := database.AutoMigrate(migrateCtx, pool); err != nil {
		slog.Error("Database migration failed", "error", err)
		os.Exit(1)
	}

	// =========================================================================
	// STEP 4: Configure Gin
	// =========================================================================
	//
	// gin.SetMode controls logging and debug features.
	// GIN_MODE=release disables debug output for production.
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// gin.New() creates a Gin engine with NO default middleware.
	// (gin.Default() adds Logger and Recovery middleware automatically,
	// but we want our own custom versions for better control.)
	r := gin.New()

	// =========================================================================
	// STEP 5: Register global middleware
	// =========================================================================
	//
	// Middleware order matters — they execute in registration order.
	// Recovery MUST be first so it wraps everything else.
	//
	// Node.js Express equivalent:
	//   app.use(express.json());
	//   app.use(cors());
	//   app.use(morgan('dev'));
	r.Use(middleware.Recovery(cfg.IsProduction()))
	r.Use(middleware.CORS(cfg.FrontendURL))
	r.Use(middleware.Logger())
	r.Use(middleware.ErrorHandler(cfg.IsProduction()))

	// Serve uploaded files statically (identical to express.static('/uploads') in Node.js)
	r.Static("/uploads", cfg.UploadDir)

	// 404 handler for unmatched routes
	r.NoRoute(middleware.NotFound())

	// =========================================================================
	// STEP 6: Wire dependencies (Dependency Injection)
	// =========================================================================
	//
	// This is the most important section to understand. In Node.js, dependencies
	// are accessed as global singletons via require(). In Go, we build them
	// explicitly and pass them as arguments. This is called the "dependency
	// injection" pattern, implemented manually without any framework.
	//
	// The dependency graph (read bottom-up — each layer depends on the one below):
	//   Handler → Service → Repository → Database Pool
	//
	// Node.js (implicit, hidden in module files):
	//   authController → prisma (global), config (global), bcrypt (global)
	//
	// Go (explicit, visible here):
	//   authHandler → authService → userRepository → pool
	//
	// Benefits: testable, auditable, no hidden coupling.

	// Repositories (database access layer)
	userRepo := repository.NewUserRepository(pool)
	examRepo := repository.NewExamRepository(pool)
	docRepo := repository.NewDocumentRepository(pool)
	chunkRepo := repository.NewDocumentChunkRepository(pool)
	quizRepo := repository.NewQuizRepository(pool)
	masteryRepo := repository.NewMasteryRepository(pool)
	pyqRepo := repository.NewPyqRepository(pool)
	mappingRepo := repository.NewTopicMappingRepository(pool)
	cardRepo := repository.NewFlashcardRepository(pool)
	planRepo := repository.NewStudyPlanRepository(pool)
	assistantRepo := repository.NewAssistantSessionRepository(pool)
	tutorRepo := repository.NewTutorSessionRepository(pool)

	// AI Client
	geminiClient := gemini.NewClient(cfg.GeminiAPIKey)

	// Services (business logic layer)
	jwtExpiry := parseDuration(cfg.JWTExpiresIn)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, jwtExpiry)
	examSvc := service.NewExamService(examRepo, geminiClient)
	ragSvc := ai.NewRAGService(chunkRepo, geminiClient)
	quizSvc := service.NewQuizService(quizRepo, examRepo, chunkRepo, geminiClient)
	masterySvc := service.NewMasteryService(masteryRepo, examRepo)
	pyqSvc := service.NewPyqService(pyqRepo, examRepo, docRepo, chunkRepo, geminiClient)
	mappingSvc := service.NewTopicMappingService(mappingRepo, examRepo, docRepo, chunkRepo, geminiClient)
	cardSvc := service.NewFlashcardService(cardRepo, examRepo, chunkRepo, geminiClient)
	planSvc := service.NewStudyPlanService(planRepo, examRepo, masteryRepo, pyqRepo)
	assistantSvc := ai.NewAssistantService(assistantRepo, examRepo, pyqRepo, ragSvc, masterySvc, planSvc, cardSvc, geminiClient)
	tutorSvc := ai.NewTutoringService(tutorRepo, chunkRepo, geminiClient)

	// AI & Background Workers
	docProcessor := worker.NewDocumentProcessor(docRepo, chunkRepo, geminiClient, mappingSvc, pyqSvc)
	workerPool := worker.NewPool(3, 100, docProcessor.Process)
	workerPool.Start()
	defer workerPool.Stop()

	docSvc := service.NewDocumentService(docRepo, examRepo, cfg.UploadDir, workerPool.Enqueue)

	// Handlers (HTTP layer)
	authHandler := handler.NewAuthHandler(authSvc)
	examHandler := handler.NewExamHandler(examSvc, geminiClient)
	docHandler := handler.NewDocumentHandler(docSvc)
	aiHandler := handler.NewAIHandler(ragSvc)
	quizHandler := handler.NewQuizHandler(quizSvc)
	masteryHandler := handler.NewMasteryHandler(masterySvc)
	pyqHandler := handler.NewPyqHandler(pyqSvc)
	mappingHandler := handler.NewMappingHandler(mappingSvc)
	cardHandler := handler.NewFlashcardHandler(cardSvc)
	planHandler := handler.NewStudyPlanHandler(planSvc)
	assistantHandler := handler.NewAssistantHandler(assistantSvc)
	tutoringHandler := handler.NewTutoringHandler(tutorSvc)

	// =========================================================================
	// STEP 7: Register routes
	// =========================================================================
	routes.Setup(
		r, cfg,
		authHandler, examHandler, docHandler, aiHandler, quizHandler,
		masteryHandler, pyqHandler, mappingHandler, cardHandler, planHandler,
		assistantHandler, tutoringHandler,
	)

	// =========================================================================
	// STEP 8: Create and start the HTTP server
	// =========================================================================
	//
	// In Node.js:
	//   const server = app.listen(config.port, () => console.log('Running on port ' + config.port))
	//
	// In Go, we create an http.Server struct explicitly instead of calling
	// r.Run(). This gives us control over timeouts and graceful shutdown.
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: r,

		// Timeouts prevent slow clients from holding connections forever.
		// Node.js has no default timeouts — this is a common production gotcha.
		ReadTimeout:    30 * time.Second,  // time to read request headers + body
		WriteTimeout:   120 * time.Second, // time to write response (generous for LLM extraction and large batch schedules)
		IdleTimeout:    120 * time.Second, // keep-alive connection idle timeout
		MaxHeaderBytes: 1 << 20,           // 1MB max header size
	}

	// Start the server in a GOROUTINE so we don't block main().
	//
	// Concept: Goroutines
	//   A goroutine is a lightweight concurrent unit of execution.
	//   "go func() { ... }()" launches a function in a new goroutine.
	//   The goroutine runs concurrently with everything else in main().
	//
	//   Think of it like: setTimeout(() => { startListening() }, 0)
	//   but properly concurrent, not just "queued after current execution".
	//
	//   Goroutines are EXTREMELY cheap — you can have millions of them.
	//   They're multiplexed onto OS threads by the Go runtime scheduler.
	go func() {
		slog.Info("Server started", "port", cfg.Port, "env", cfg.Env)

		// ListenAndServe blocks until the server stops.
		// http.ErrServerClosed is returned when Shutdown() is called — that's expected.
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// =========================================================================
	// STEP 9: Graceful Shutdown
	// =========================================================================
	//
	// In Node.js (the broken version):
	//   process.on('unhandledRejection', (err) => server.close(() => process.exit(1)));
	// This is NOT graceful — it doesn't wait for in-flight requests.
	//
	// In Go, we:
	//   1. Listen for OS signals (Ctrl+C = SIGINT, kill = SIGTERM)
	//   2. When signal arrives, call srv.Shutdown(ctx) which:
	//      - Stops accepting NEW connections
	//      - Waits for existing in-flight requests to finish
	//      - Times out after ctx expires (30 seconds)
	//   3. All deferred cleanup (pool.Close()) runs after Shutdown returns
	//
	// This is PRODUCTION-GRADE graceful shutdown — something Node.js requires
	// significant third-party tooling or careful manual implementation to achieve.

	// Concept: Channels
	//   Channels are Go's mechanism for safe communication between goroutines.
	//   Think of a channel as a typed, goroutine-safe pipe.
	//
	//   make(chan os.Signal, 1) creates a buffered channel that holds 1 os.Signal.
	//   The buffer of 1 is important: signal.Notify sends to the channel even if
	//   nobody is reading yet. Without the buffer, the signal could be dropped.
	quit := make(chan os.Signal, 1)

	// signal.Notify registers our channel to receive SIGINT (Ctrl+C) and SIGTERM (kill).
	// SIGTERM is what Docker/Kubernetes sends when stopping a container.
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block main() here until a signal arrives.
	// <- quit READS from the channel. This blocks until something is sent to it.
	// signal.Notify sends to it when the OS delivers SIGINT or SIGTERM.
	<-quit

	slog.Info("Shutdown signal received, gracefully shutting down...")

	// Give in-flight requests 30 seconds to complete.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// srv.Shutdown() tells the server to:
	//   1. Stop accepting new connections immediately
	//   2. Wait for all active handlers to return
	//   3. Return when all handlers are done (or timeout expires)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited gracefully")
	// At this point, defer pool.Close() runs — connections are released cleanly
}

// parseDuration converts a JWT expiry string like "7d", "24h", "30m" to time.Duration.
// In Node.js, jsonwebtoken accepted "7d" as a string.
// In Go, time.ParseDuration understands "24h", "30m", "90s" but NOT "7d".
// We handle the "d" suffix ourselves.
func parseDuration(s string) time.Duration {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		// Try to parse the numeric part
		var days int
		if _, err := fmt.Sscanf(s[:len(s)-1], "%d", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	// Fall back to standard Go duration parsing ("24h", "30m", "90s")
	d, err := time.ParseDuration(s)
	if err != nil {
		// Default to 7 days if parsing fails
		return 7 * 24 * time.Hour
	}
	return d
}

