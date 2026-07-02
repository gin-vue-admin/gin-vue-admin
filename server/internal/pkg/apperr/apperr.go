// Package apperr 定义应用层错误：统一映射到 RFC 7807 ProblemDetail。
// 业务代码返回 *apperr.Error，handler 层用 apperr.Write 写出标准错误响应。
package apperr

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gva/internal/pkg/log"
	"gva/internal/pkg/response"
)

// Error 应用层错误，携带 HTTP 状态码与 ProblemDetail 字段。
type Error struct {
	Status int                 // HTTP 状态码
	Code   string              // 应用层错误码（机器可读）
	Title  string              // 简短摘要（空则按 Status 推断）
	Detail string              // 具体说明
	Errors map[string][]string // 字段级错误（表单校验）
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return e.Code
}

// New 构造一个应用错误。
func New(status int, code, title, detail string) *Error {
	return &Error{Status: status, Code: code, Title: title, Detail: detail}
}

// 常用预置错误。
func Unauthorized(detail ...string) *Error {
	d := "未授权"
	if len(detail) > 0 {
		d = detail[0]
	}
	return &Error{Status: 401, Code: "UNAUTHORIZED", Title: "Unauthorized", Detail: d}
}

func Forbidden(detail ...string) *Error {
	d := "禁止访问"
	if len(detail) > 0 {
		d = detail[0]
	}
	return &Error{Status: 403, Code: "FORBIDDEN", Title: "Forbidden", Detail: d}
}

func NotFound(detail ...string) *Error {
	d := "资源不存在"
	if len(detail) > 0 {
		d = detail[0]
	}
	return &Error{Status: 404, Code: "NOT_FOUND", Title: "Not Found", Detail: d}
}

func BadRequest(code, detail string) *Error {
	return &Error{Status: 400, Code: code, Title: "Bad Request", Detail: detail}
}

// Conflict 409（如唯一约束冲突）。
func Conflict(detail string) *Error {
	return &Error{Status: 409, Code: "CONFLICT", Title: "Conflict", Detail: detail}
}

// Validation 422 校验失败（请求体字段不合规）。
func Validation(detail string, errs map[string][]string) *Error {
	return &Error{Status: 422, Code: "VALIDATION_ERROR", Title: "Unprocessable Entity", Detail: detail, Errors: errs}
}

// As 包装 errors.As，便于外部判断。
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// Write 统一将错误写出为 HTTP 响应：应用错误按其 Status/Code 写 ProblemDetail，
// 未知错误记日志后回 500，不向前端泄漏内部细节。
func Write(c *gin.Context, err error) {
	if e, ok := As(err); ok {
		response.Problem(c, e.Status, e.Title, e.Detail,
			response.WithCode(e.Code),
			response.WithErrors(e.Errors),
		)
		return
	}
	log.S.Errorw("未预期错误", "err", err, "path", c.Request.URL.Path)
	response.Problem(c, 500, "", "服务暂时不可用",
		response.WithCode("INTERNAL_ERROR"),
	)
}
