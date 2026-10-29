package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"bank-app-ioc/backend/internal/telemetry"
	"errors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"bank-app-ioc/backend/config"
	"bank-app-ioc/backend/database"
	_ "bank-app-ioc/backend/docs"
	"bank-app-ioc/backend/internal/account"
	"bank-app-ioc/backend/internal/admin"
	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/internal/beneficiary"
	"bank-app-ioc/backend/internal/httpx"
	"bank-app-ioc/backend/internal/middleware"
	"bank-app-ioc/backend/internal/operations"
	"bank-app-ioc/backend/internal/transfer"
	"bank-app-ioc/backend/repository"
)

// @title Online Banking Mini Application API
// @version 1.0
// @description Online banking API. Single currency INR; amounts are decimal strings with 2 places. Authentication uses a session cookie.
// @BasePath /api
// @securityDefinitions.apikey SessionAuth
// @in cookie
// @name session_id
// @securityDefinitions.apikey CSRFToken
// @in header
// @name X-CSRF-Token
func main() {
	telemetry.InitLogger()
	appConfig, err := config.Load()
	if err != nil {
		log.Fatal("configuration error")
	}

	pool, err := database.NewPool(context.Background(), appConfig.DatabaseURL)
	if err != nil {
		log.Fatal("database connection failed")
	}
	defer pool.Close()
	shutdownTelemetry, err := telemetry.Setup(context.Background(), "bank-api", "1.0.0")
	if err != nil {
		log.Fatal("telemetry setup failed")
	}
	sessionRepository := repository.NewSessionRepository(pool)
	userRepository := repository.NewUserRepository(pool)
	authService := auth.NewService(userRepository, sessionRepository, appConfig.SessionIdleTimeout, appConfig.SessionMaxLifetime, appConfig.PasswordMinLength)
	auditService := audit.NewService(repository.NewAuditRepository(pool))
	router := gin.New()
	if err := router.SetTrustedProxies(appConfig.TrustedProxies); err != nil {
		log.Fatal("invalid trusted proxies configuration")
	}
	authLimiter := middleware.NewRateLimiterWithClock(5, time.Minute, time.Now, auditService)
	defer authLimiter.Stop()
	router.Use(
		otelgin.Middleware("bank-api", otelgin.WithFilter(func(r *http.Request) bool {
			return r.URL.Path != "/health" && r.URL.Path != "/ready" // don't trace probes
		})),
		middleware.CorrelationID(), middleware.Logging(), telemetry.HTTPMetrics(), middleware.Recovery(),
		middleware.SecurityHeaders(), middleware.ErrorHandler(), authLimiter.Handler(),
	)
	router.NoRoute(func(c *gin.Context) { httpx.Fail(c, apperr.NotFound()) })
	if appConfig.SwaggerEnabled {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	configRepository := repository.NewConfigRepository(pool)
	adminHandler := admin.NewHandler(admin.NewService(
		repository.NewAdminRepository(pool),
		repository.NewAssignmentRepository(pool),
		configRepository,
		repository.NewAuditQueryRepository(pool),
		appConfig.PasswordMinLength,
	))
	authHandler := auth.NewHandler(authService, auditService, appConfig.CSRFSecret)
	accountHandler := account.NewHandler(account.NewService(repository.NewAccountRepository(pool)), auditService)
	beneficiaryHandler := beneficiary.NewHandler(beneficiary.NewService(repository.NewBeneficiaryRepository(pool)), auditService)
	transferHandler := transfer.NewHandler(transfer.NewServiceFromConfig(repository.NewTransferRepository(pool), appConfig).WithLimitProvider(configRepository.MaxTransferPaise), auditService)
	operationsHandler := operations.NewHandler(operations.NewService(repository.NewAccountRepository(pool), repository.NewAssignmentRepository(pool), repository.NewTransferRepository(pool)), auditService)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	var draining atomic.Bool
	router.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if draining.Load() || pool.Ping(ctx) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	router.POST("/api/auth/signup", authHandler.Signup)
	router.POST("/api/auth/login", authHandler.Login)
	protected := router.Group("/api", middleware.Authenticate(authService), middleware.CSRF(appConfig.CSRFSecret))
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/me", authHandler.Me)
	customer := protected.Group("", middleware.CustomerOnly(auditService))
	transferLimiter := middleware.NewTransferRateLimiter(appConfig.TransferRateLimit, time.Minute, auditService)
	defer transferLimiter.Stop()
	customer.GET("/accounts", accountHandler.List)
	customer.GET("/accounts/:accountId", accountHandler.Get)
	customer.GET("/accounts/:accountId/transactions", accountHandler.Transactions)
	customer.POST("/beneficiaries", beneficiaryHandler.Create)
	customer.GET("/beneficiaries", beneficiaryHandler.List)
	customer.PATCH("/beneficiaries/:beneficiaryId/status", beneficiaryHandler.UpdateStatus)
	customer.POST("/transfers", transferLimiter.Handler(), transferHandler.Create)
	customer.GET("/transfers", transferHandler.List)
	protected.GET("/transfers/:referenceNumber", transferHandler.Get)
	ops := protected.Group("/operations", middleware.OperationsOnly(auditService))
	ops.GET("/accounts", operationsHandler.ListAccounts)
	ops.GET("/accounts/:accountId/transactions", operationsHandler.Transactions)
	ops.GET("/transfers", operationsHandler.ListTransfers)
	ops.PATCH("/transfers/:reference/review", operationsHandler.ReviewTransfer)
	ops.POST("/accounts/:accountId/freeze", operationsHandler.Freeze)
	ops.POST("/accounts/:accountId/unfreeze", operationsHandler.Unfreeze)
	adminRoutes := protected.Group("/admin", middleware.AdminOnly(auditService))
	adminRoutes.POST("/users", adminHandler.CreateUser)
	adminRoutes.PATCH("/users/:userId/status", adminHandler.SetStatus)
	adminRoutes.PATCH("/users/:userId/role", adminHandler.ChangeRole)
	adminRoutes.POST("/users/:userId/reset-password", adminHandler.ResetPassword)
	adminRoutes.POST("/account-assignments", adminHandler.AssignAccount)
	adminRoutes.GET("/config/transaction-limit", adminHandler.GetLimit)
	adminRoutes.PATCH("/config/transaction-limit", adminHandler.SetLimit)
	adminRoutes.GET("/audit-events", adminHandler.SearchAudit)

	srv := &http.Server{
		Addr:              ":" + appConfig.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second, // blocks slowloris
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server listening", "port", appConfig.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			stop() // fall into the shutdown path below
		}
	}()

	<-ctx.Done() // Ctrl+C, SIGTERM (docker stop / deploy), or server failure
	slog.Info("shutting down")
	draining.Store(true) // /ready flips to 503 so a load balancer drains us

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil { // stop accepting, finish in-flight transfers
		slog.Error("graceful shutdown failed", "error", err)
	}
	if err := shutdownTelemetry(shutdownCtx); err != nil { // flush buffered spans/metrics
		slog.Error("telemetry flush failed", "error", err)
	}
	// deferred limiter Stop() and pool.Close() run when main returns, AFTER requests finish
}
