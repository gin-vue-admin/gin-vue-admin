// Package middleware 操作日志中间件：自动记录写操作（POST/PUT/PATCH/DELETE）。
package middleware

import (
	"bytes"
	"context"
	"io"
	"time"

	"gva/internal/model"
	"gva/internal/pkg/async"
	"gva/internal/pkg/log"
	"gva/internal/repository"

	"github.com/gin-gonic/gin"
)

// isWriteMethod 仅记录写操作（GET/HEAD/OPTIONS 不记录）。
func isWriteMethod(m string) bool {
	switch m {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}

// OperationLog 操作日志中间件：
//   - c.Next() 前读取 body（用于 params），再重置供后续 handler 绑定
//   - c.Next() 后收集 method/path/ip/username/HTTP 状态/耗时，异步写入
//
// 异步用独立 context.Background()（请求 ctx 在请求结束后取消），不阻塞响应。
// CreatedBy 审计字段因独立 ctx 无 userID 而为零值——日志的 username 字段已记录操作者，足够。
func OperationLog(repo repository.OperationLogRepository, runner async.Runner) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isWriteMethod(c.Request.Method) {
			c.Next()
			return
		}
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		start := time.Now()
		c.Next()
		duration := time.Since(start).Milliseconds()

		usernameStr := ""
		if v, ok := c.Get(ContextKeyUsername); ok {
			if s, ok := v.(string); ok {
				usernameStr = s
			}
		}
		status := "success"
		if c.Writer.Status() >= 400 {
			status = "failed"
		}
		params := string(bodyBytes)
		if len(params) > 2000 { // 截断防止超大 body
			params = params[:2000]
		}

		entry := &model.OperationLog{
			Username:  usernameStr,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Params:    params,
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Status:    status,
			HTTPCode:  c.Writer.Status(),
			Duration:  duration,
		}
		runner.Go(func() {
			if err := repo.Create(context.Background(), entry); err != nil {
				log.S.Warnw("操作日志写入失败", "path", entry.Path, "err", err)
			}
		})
	}
}
