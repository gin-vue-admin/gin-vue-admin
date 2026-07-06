package middleware

import (
	"time"

	"gva/internal/pkg/log"
	"gva/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 记录每个请求的访问日志（方法/路径/状态/耗时/traceId）。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.L.Info("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int("size", c.Writer.Size()),
			zap.Duration("latency", time.Since(start)),
			zap.String("traceId", c.GetString(response.ContextKeyTraceID)),
		)
	}
}
