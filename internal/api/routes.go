package api

import (
	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/leeaandrob/claudex/internal/api/handlers"
	"github.com/leeaandrob/claudex/internal/api/middleware"
	"github.com/leeaandrob/claudex/internal/claude"
	"github.com/leeaandrob/claudex/internal/converter"
	"github.com/leeaandrob/claudex/internal/mcp"
	"github.com/leeaandrob/claudex/internal/observability"
)

// RegisterRoutes registers all API routes.
func RegisterRoutes(app *fiber.App, logger *observability.Logger, metrics *observability.Metrics, executor *claude.Executor, mcpManager *mcp.Manager) {
	// Add OpenTelemetry middleware
	app.Use(otelfiber.Middleware(
		otelfiber.WithServerName("openai-claude-proxy"),
	))

	// Add request ID middleware
	app.Use(middleware.RequestID())

	// Add logging middleware
	app.Use(middleware.Logging(logger))

	// Health check endpoints (no middleware)
	app.Use(healthcheck.New(healthcheck.Config{
		LivenessProbe: func(c *fiber.Ctx) bool {
			return true
		},
		LivenessEndpoint: "/livez",
		ReadinessProbe: func(c *fiber.Ctx) bool {
			// Check if Claude CLI is available
			return executor.IsAvailable()
		},
		ReadinessEndpoint: "/readyz",
	}))

	// Prometheus metrics endpoint
	app.Get("/metrics", func(c *fiber.Ctx) error {
		fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())(c.Context())
		return nil
	})

	// Create chat completions handler
	parser := claude.NewParser()
	conv := converter.NewConverter()
	chatHandler := handlers.NewChatCompletionsHandler(executor, parser, conv, mcpManager, metrics, logger)

	// API routes
	v1 := app.Group("/v1")
	v1.Post("/chat/completions", chatHandler.Handle)

	// MCP tools endpoint (for debugging/discovery)
	v1.Get("/mcp/tools", func(c *fiber.Ctx) error {
		tools := mcpManager.GetAllTools()
		return c.JSON(fiber.Map{
			"tools": tools,
			"count": len(tools),
		})
	})

	// MCP servers endpoint (for debugging/discovery)
	v1.Get("/mcp/servers", func(c *fiber.Ctx) error {
		clients := mcpManager.GetClients()
		return c.JSON(fiber.Map{
			"servers": clients,
			"count":   len(clients),
		})
	})
}
