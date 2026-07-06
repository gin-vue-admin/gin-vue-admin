package handler

import (
	"gva/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查端点。
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// Health GET /api/health
// @Summary      健康检查
// @Description  探活端点，无需鉴权
// @Tags         system
// @Produce      json
// @Success      200  {object}  response.ApiResult
// @Failure      500  {object}  response.ProblemDetail
// @Router       /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	response.Success(c, gin.H{"status": "up"})
}
