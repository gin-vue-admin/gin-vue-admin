package handler

import (
	"github.com/gin-gonic/gin"
	"gva/internal/pkg/response"
)

// HealthHandler 健康检查端点。
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// Health GET /api/health
func (h *HealthHandler) Health(c *gin.Context) {
	response.Success(c, gin.H{"status": "up"})
}
