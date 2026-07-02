package response

import "github.com/gin-gonic/gin"

// HTTP 状态码 → 标准标题（对齐前端 problem.ts 的 HTTP_STATUS_TITLE）。
var httpStatusTitle = map[int]string{
	400: "Bad Request",
	401: "Unauthorized",
	403: "Forbidden",
	404: "Not Found",
	405: "Method Not Allowed",
	406: "Not Acceptable",
	409: "Conflict",
	415: "Unsupported Media Type",
	422: "Unprocessable Entity",
	429: "Too Many Requests",
	500: "Internal Server Error",
	502: "Bad Gateway",
	503: "Service Unavailable",
}

// ProblemDetail RFC 7807 错误响应结构。
type ProblemDetail struct {
	Type     string              `json:"type"`           // 问题类型 URI
	Title    string              `json:"title"`          // 简短摘要
	Status   int                 `json:"status"`         // HTTP 状态码
	Detail   string              `json:"detail"`         // 具体说明
	Instance string              `json:"instance,omitempty"`
	Code     string              `json:"code,omitempty"`     // 应用层错误码（机器可读）
	Errors   map[string][]string `json:"errors,omitempty"`   // 字段级错误（表单校验）
	TraceID  string              `json:"traceId,omitempty"`
}

// ProblemOption 用于细粒度定制 ProblemDetail。
type ProblemOption func(*ProblemDetail)

// WithCode 设置应用层错误码。
func WithCode(code string) ProblemOption {
	return func(p *ProblemDetail) { p.Code = code }
}

// WithInstance 设置资源 URI。
func WithInstance(inst string) ProblemOption {
	return func(p *ProblemDetail) { p.Instance = inst }
}

// WithErrors 设置字段级校验错误。
func WithErrors(errs map[string][]string) ProblemOption {
	return func(p *ProblemDetail) { p.Errors = errs }
}

// titleOf 取状态码对应标题，缺省回退到 "HTTP <status>"。
func titleOf(status int) string {
	if t, ok := httpStatusTitle[status]; ok {
		return t
	}
	return ""
}

// Problem 写出错误响应并终止后续处理。title 为空时按 status 自动推断。
func Problem(c *gin.Context, status int, title, detail string, opts ...ProblemOption) {
	if title == "" {
		title = titleOf(status)
	}
	p := ProblemDetail{
		Type:    "about:blank",
		Title:   title,
		Status:  status,
		Detail:  detail,
		TraceID: traceIDFrom(c),
		Instance: c.Request.URL.Path,
	}
	for _, opt := range opts {
		opt(&p)
	}
	c.AbortWithStatusJSON(status, p)
}
