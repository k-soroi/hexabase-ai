package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hexabase/hexabase-ai/api/internal/shared/api/routes"
	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
	"github.com/hexabase/hexabase-ai/api/internal/shared/db"
	"github.com/hexabase/hexabase-ai/api/internal/shared/infrastructure/wire"
	"github.com/hexabase/hexabase-ai/api/internal/shared/logging"
	"github.com/hexabase/hexabase-ai/api/internal/shared/middleware"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Initialize logger
	logger := logging.NewLogger()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	// Connect to database
	dbConfig := &db.DatabaseConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	database, err := db.ConnectDatabase(dbConfig)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Initialize Gin router
	if cfg.Server.Host == "0.0.0.0" && os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.StructuredLogger(logger))

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Health check endpoint (basic - doesn't require DB)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "0.1.0",
		})
	})

	// Readiness check endpoint (requires DB)
	router.GET("/ready", func(c *gin.Context) {
		sqlDB, err := database.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database connection error",
			})
			return
		}
		
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready", 
				"error":  "database ping failed",
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Initialize Kubernetes client
	var k8sConfig *rest.Config
	var k8sClient kubernetes.Interface
	var dynamicClient dynamic.Interface

	// Try to use in-cluster config first
	k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
		}
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			logger.Warn("Failed to initialize Kubernetes client", "error", err)
		}
	}

	if k8sConfig != nil {
		k8sClient, err = kubernetes.NewForConfig(k8sConfig)
		if err != nil {
			logger.Warn("Failed to create Kubernetes client", "error", err)
		}

		dynamicClient, err = dynamic.NewForConfig(k8sConfig)
		if err != nil {
			logger.Warn("Failed to create dynamic Kubernetes client", "error", err)
		}
	}

	// Initialize application with dependency injection
	app, err := wire.InitializeApp(cfg, database, k8sClient, dynamicClient, k8sConfig, logger)
	if err != nil {
		logger.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	// Setup routes
	routes.SetupRoutes(router, app)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server",
			"address", srv.Addr,
			"environment", gin.Mode(),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited")
}