// Package handler 请求/响应结构与公共辅助。
package handler

// batchDeleteReq 批量删除请求：ids 数组（至少 1 个）。
// 各资源 handler 共用此结构，避免 8 处重复定义；类型别名见各 handler 文件。
type batchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}
