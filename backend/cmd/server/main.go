package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

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
	appConfig, err := config.Load()
	if err != nil {
		log.Fatal("configuration error")
	}

	pool, err := database.NewPool(context.Background(), appConfig.DatabaseURL)
	if err != nil {
		log.Fatal("database connection failed")
	}
	defer pool.Close()

	sessionRepository := repository.NewSessionRepository(pool)
	userRepository := repository.NewUserRepository(pool)
	authService := auth.NewService(userRepository, sessionRepository, appConfig.SessionIdleTimeout, appConfig.SessionMaxLifetime, appConfig.PasswordMinLength)
	auditService := audit.NewService(repository.NewAuditRepository(pool))
	router := gin.New()
	if err := router.SetTrustedProxies(appConfig.TrustedProxies); err != nil {
		log.Fatal("invalid trusted proxies configuration")
	}
	router.Use(middleware.CorrelationID(), middleware.Logging(), middleware.Recovery(), middleware.SecurityHeaders(), middleware.ErrorHandler(), middleware.NewRateLimiterWithClock(5, time.Minute, time.Now, auditService).Handler())
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

	log.Printf("server listening on :%s", appConfig.Port)
	if err := router.Run(":" + appConfig.Port); err != nil {
		log.Fatal("server stopped unexpectedly")
	}
}
