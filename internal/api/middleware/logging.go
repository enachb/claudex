package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/leeaandrob/claudex/internal/observability"
)

// Logging creates a middleware that logs requests.
func Logging(logger *observability.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		requestID := GetRequestID(c)

		// Log request start
		logger.Info("request started",
			"method", c.Method(),
			"path", c.Path(),
			"request_id", requestID,
			"ip", c.IP(),
		)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		logger.Info("request completed",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", duration.Milliseconds(),
			"request_id", requestID,
		)

		return err
	}
}
