// Package middleware 操作日志中间件：自动记录写操作（POST/PUT/PATCH/DELETE）。
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
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

// sensitiveFields 操作日志中需脱敏的 JSON 字段（值替换为 ***），防止明文落库。
// 仅处理顶层字段（登录/改密等请求体均为扁平结构）。
var sensitiveFields = []string{"password", "oldPassword", "newPassword", "secret", "token", "captcha"}

// redactParams 对 JSON 请求体做敏感字段脱敏后截断；非 JSON 仅截断。
func redactParams(contentType, body string) string {
	if strings.Contains(contentType, "application/json") {
		var m map[string]any
		if json.Unmarshal([]byte(body), &m) == nil {
			for _, f := range sensitiveFields {
				if _, ok := m[f]; ok {
					m[f] = "***"
				}
			}
			if b, err := json.Marshal(m); err == nil {
				body = string(b)
			}
		}
	}
	if len(body) > 2000 { // 截断防止超大 body
		body = body[:2000]
	}
	return body
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
		params := redactParams(c.Request.Header.Get("Content-Type"), string(bodyBytes))

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
