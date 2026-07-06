package middleware

import (
	"gva/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceID 为每个请求注入/透传 traceId，写入上下文与响应头。
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(response.ContextKeyTraceID, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}
