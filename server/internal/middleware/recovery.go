package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gva/internal/pkg/log"
	"gva/internal/pkg/response"
)

// Recovery 捕获 panic，记录日志并返回 500 ProblemDetail，避免进程崩溃。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.L.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
				)
				response.Problem(c, 500, "", "服务器内部错误")
			}
		}()
		c.Next()
	}
}
