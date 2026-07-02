// Package response 统一 HTTP 响应封装，对齐前端 RFC 7807 契约。
// 成功：HTTP 200 + { code:0, data, msg, traceId? }
// 失败：HTTP 4xx/5xx + ProblemDetail（见 problem.go）
package response

import "github.com/gin-gonic/gin"

// ContextKeyTraceID 是存放在 gin.Context 中的 traceId 键。
const ContextKeyTraceID = "traceId"

// CodeSuccess 业务成功码。
const CodeSuccess = 0

// ApiResult 业务成功响应结构（HTTP 200）。
type ApiResult struct {
	Code    int    `json:"code"`
	Data    any    `json:"data"`
	Msg     string `json:"msg"`
	TraceID string `json:"traceId,omitempty"`
}

// traceIDFrom 从上下文中读取 traceId（由 TraceID 中间件注入）。
func traceIDFrom(c *gin.Context) string {
	if v, ok := c.Get(ContextKeyTraceID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Success 写出成功响应。msg 可选，默认 "ok"。
func Success(c *gin.Context, data any, msg ...string) {
	m := "ok"
	if len(msg) > 0 && msg[0] != "" {
		m = msg[0]
	}
	c.JSON(200, ApiResult{
		Code:    CodeSuccess,
		Data:    data,
		Msg:     m,
		TraceID: traceIDFrom(c),
	})
}
