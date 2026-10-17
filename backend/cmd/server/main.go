package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"bank-app-ioc/backend/config"
	"bank-app-ioc/backend/database"
	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/internal/httpx"
	"bank-app-ioc/backend/internal/middleware"
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
	authService := auth.NewService(userRepository, sessionRepository, appConfig.SessionIdleTimeout, appConfig.SessionMaxLifetime, appConfig.PasswordMinLength)
	auditService := audit.NewService(repository.NewAuditRepository(pool))
	authHandler := auth.NewHandler(authService, auditService, appConfig.CSRFSecret)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/api/auth/signup", authHandler.Signup)
	router.POST("/api/auth/login", authHandler.Login)
	protected := router.Group("/api", middleware.Authenticate(authService), middleware.CSRF(appConfig.CSRFSecret))
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/me", authHandler.Me)

	log.Printf("server listening on :%s", appConfig.Port)
	if err := router.Run(":" + appConfig.Port); err != nil {
		log.Fatal("server stopped unexpectedly")
	}
}
