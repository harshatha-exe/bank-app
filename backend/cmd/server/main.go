package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"bank-app-ioc/backend/config"
	"bank-app-ioc/backend/database"
	"bank-app-ioc/backend/internal/account"
	"bank-app-ioc/backend/internal/admin"
	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/internal/beneficiary"
	"bank-app-ioc/backend/internal/httpx"
	"bank-app-ioc/backend/internal/middleware"
	"bank-app-ioc/backend/internal/transfer"
	"bank-app-ioc/backend/repository"
)

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

	router := gin.New()
	router.Use(middleware.CorrelationID(), middleware.Logging(), middleware.Recovery(), middleware.SecurityHeaders(), middleware.ErrorHandler(), middleware.NewRateLimiter(5, time.Minute))
	router.NoRoute(func(c *gin.Context) { httpx.Fail(c, apperr.NotFound()) })
	sessionRepository := repository.NewSessionRepository(pool)
	userRepository := repository.NewUserRepository(pool)
	authService := auth.NewService(userRepository, sessionRepository, appConfig.SessionIdleTimeout, appConfig.SessionMaxLifetime, appConfig.PasswordMinLength)
	auditService := audit.NewService(repository.NewAuditRepository(pool))
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
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/api/auth/signup", authHandler.Signup)
	router.POST("/api/auth/login", authHandler.Login)
	protected := router.Group("/api", middleware.Authenticate(authService), middleware.CSRF(appConfig.CSRFSecret))
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/me", authHandler.Me)
	customer := protected.Group("", middleware.CustomerOnly(auditService))
	customer.GET("/accounts", accountHandler.List)
	customer.GET("/accounts/:accountId", accountHandler.Get)
	customer.GET("/accounts/:accountId/transactions", accountHandler.Transactions)
	customer.POST("/beneficiaries", beneficiaryHandler.Create)
	customer.GET("/beneficiaries", beneficiaryHandler.List)
	customer.PATCH("/beneficiaries/:beneficiaryId/status", beneficiaryHandler.UpdateStatus)
	customer.POST("/transfers", transferHandler.Create)
	customer.GET("/transfers", transferHandler.List)
	protected.GET("/transfers/:referenceNumber", transferHandler.Get)
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
