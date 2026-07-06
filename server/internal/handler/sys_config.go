// Package handler 系统参数配置端点：列表/详情（按 id 与按 key）/增改删。
// 注：无 Export/批删（参数配置数据量小且语义上不宜批删，遵循 YAGNI）。
package handler

import (
	"strconv"

	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/pkg/response"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
)

// sysConfigUpsertReq 增改请求体。
type sysConfigUpsertReq struct {
	ConfigKey   string `json:"configKey" binding:"required,max=128"`
	ConfigValue string `json:"configValue"`
	ConfigName  string `json:"configName" binding:"max=128"`
	Remark      string `json:"remark"`
	Type        string `json:"type"`
}

// SysConfigHandler 系统参数配置处理器。
type SysConfigHandler struct {
	svc *service.SysConfigService
}

func NewSysConfigHandler(svc *service.SysConfigService) *SysConfigHandler {
	return &SysConfigHandler{svc: svc}
}

// List GET /api/system/config
// @Summary      系统配置分页列表
// @Tags         sys-config
// @Produce      json
// @Security     BearerAuth
// @Param        page    query int    false "页码" default(1)
// @Param        size    query int    false "每页" default(10)
// @Param        keyword query string false "key/name 关键词"
// @Success      200  {object} response.ApiResult
// @Router       /system/config [get]
func (h *SysConfigHandler) List(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	res, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Get GET /api/system/config/:id
// @Summary      系统配置详情（按 ID）
// @Tags         sys-config
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/config/{id} [get]
func (h *SysConfigHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	e, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e)
}

// GetByKey GET /api/system/config/key/:key  按 key 取配置（前端按 key 直取场景）。
// @Summary      按 key 取配置
// @Tags         sys-config
// @Produce      json
// @Security     BearerAuth
// @Param        key  path string true "配置 key"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/config/key/{key} [get]
func (h *SysConfigHandler) GetByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		apperr.Write(c, apperr.Validation("无效的 key", nil))
		return
	}
	e, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e)
}

// GetPublic GET /api/system/config/public（无需鉴权，返回客户端可见配置：site_title 等）
// @Summary      公开配置
// @Description  返回白名单配置（site_title/login_captcha_enabled/default_page_size），前端启动拉取
// @Tags         sys-config
// @Produce      json
// @Success      200  {object} response.ApiResult
// @Router       /system/config/public [get]
func (h *SysConfigHandler) GetPublic(c *gin.Context) {
	response.Success(c, h.svc.GetPublic())
}

// Create POST /api/system/config
// @Summary      创建系统配置
// @Tags         sys-config
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body sysConfigUpsertReq true "配置信息"
// @Success      200  {object} response.ApiResult
// @Failure      409  {object} response.ProblemDetail "key 已存在"
// @Router       /system/config [post]
func (h *SysConfigHandler) Create(c *gin.Context) {
	var req sysConfigUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	e := &model.SysConfig{
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		ConfigName:  req.ConfigName,
		Remark:      req.Remark,
		Type:        req.Type,
	}
	if err := h.svc.Create(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "创建成功")
}

// Update PUT /api/system/config/:id
// @Summary      更新系统配置
// @Tags         sys-config
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "ID"
// @Param        request body sysConfigUpsertReq true "配置信息"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/config/{id} [put]
func (h *SysConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req sysConfigUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	e, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	e.ConfigKey = req.ConfigKey
	e.ConfigValue = req.ConfigValue
	e.ConfigName = req.ConfigName
	e.Remark = req.Remark
	e.Type = req.Type
	if err := h.svc.Update(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "更新成功")
}

// Delete DELETE /api/system/config/:id
// @Summary      删除系统配置
// @Tags         sys-config
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/config/{id} [delete]
func (h *SysConfigHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}
