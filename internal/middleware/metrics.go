package middleware

import (
	"strconv"
	"time"

	"menu-service/pkg/metrics"

	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		metrics.RecordHTTPRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), time.Since(start))
	}
}

func MetricsHandler(namespace, subsystem string) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload := metrics.RenderPrometheus(namespace, subsystem)
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.Header("Content-Length", strconv.Itoa(len(payload)))
		c.String(200, payload)
	}
}
