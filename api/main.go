package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"finopsbridge/api/internal/config"
	"finopsbridge/api/internal/database"
	"finopsbridge/api/internal/handlers"
	"finopsbridge/api/internal/middleware"
	"finopsbridge/api/internal/opa"
	"finopsbridge/api/internal/worker"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize OPA
	opaEngine, err := opa.Initialize(cfg.OPADir)
	if err != nil {
		log.Fatalf("Failed to initialize OPA: %v", err)
	}
	defer opaEngine.Close()

	// Start OPA hot reload watcher
	go opaEngine.WatchForChanges()

	// Initialize handlers
	h := handlers.New(db, opaEngine, cfg)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: handlers.ErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// API routes
	api := app.Group("/api")
	api.Use(middleware.ClerkAuth(cfg.ClerkSecretKey))

	// Waitlist (public)
	app.Post("/api/waitlist", h.CreateWaitlistEntry)

	// Dashboard
	api.Get("/dashboard/stats", h.GetDashboardStats)

	// Policies
	api.Get("/policies", h.ListPolicies)
	api.Get("/policies/:id", h.GetPolicy)
	api.Post("/policies", h.CreatePolicy)
	api.Patch("/policies/:id", h.UpdatePolicy)
	api.Delete("/policies/:id", h.DeletePolicy)

	// Cloud Providers
	api.Get("/cloud-providers", h.ListCloudProviders)
	api.Get("/cloud-providers/:id", h.GetCloudProvider)
	api.Post("/cloud-providers", h.CreateCloudProvider)
	api.Delete("/cloud-providers/:id", h.DeleteCloudProvider)

	// Activity Log
	api.Get("/activity", h.ListActivityLogs)

	// Webhooks
	api.Get("/webhooks", h.ListWebhooks)
	api.Post("/webhooks", h.CreateWebhook)
	api.Delete("/webhooks/:id", h.DeleteWebhook)

	// Start enforcement worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	enforcementWorker := worker.NewEnforcementWorker(db, opaEngine, cfg)
	go enforcementWorker.Start(ctx, 5*time.Minute)

	// Start server
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()
	app.Shutdown()
}

