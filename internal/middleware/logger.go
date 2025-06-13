package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLogger logs incoming requests and adds a request_id for tracing.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Set("request_ip", c.ClientIP())

		// Process request
		c.Next()

		// Log details after request is processed
		latency := time.Since(start)
		log.Printf(
			"[Request] ID: %s | Status: %d | Latency: %s | Method: %s | Path: %s | IP: %s",
			requestID,
			c.Writer.Status(),
			latency,
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
		)
	}
}
