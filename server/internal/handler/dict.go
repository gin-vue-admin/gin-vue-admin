// Package handler 字典管理端点（三级：分类/字典/字典项，各 CRUD + 导出）。
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

// dictBatchDeleteReq 三级共用的批量删除请求体。
type dictBatchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// parseIDFromCtx 解析路径 :id，失败写 422。
func parseIDFromCtx(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return 0, false
	}
	return uint(id), true
}

// parseIDs 把前端 string id 数组转 []uint，失败写 422。
func parseIDs(c *gin.Context, ss []string) ([]uint, bool) {
	ids := make([]uint, 0, len(ss))
	for _, s := range ss {
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			apperr.Write(c, apperr.Validation("无效的 id: "+s, nil))
			return nil, false
		}
		ids = append(ids, uint(id))
	}
	return ids, true
}

// ==================== Level 1: 字典分类 ====================

// DictCategoryHandler 字典分类 HTTP 端点。
type DictCategoryHandler struct {
	svc *service.DictCategoryService
}

func NewDictCategoryHandler(svc *service.DictCategoryService) *DictCategoryHandler {
	return &DictCategoryHandler{svc: svc}
}

// List GET /api/dict/categories
// @Summary      字典分类分页列表
// @Tags         dict-category
// @Produce      json
// @Security     BearerAuth
// @Param        page    query int    false "页码" default(1)
// @Param        size    query int    false "每页" default(10)
// @Param        keyword query string false "名称/编码关键词"
// @Param        status  query string false "状态"
// @Success      200  {object} response.ApiResult
// @Router       /dict/categories [get]
func (h *DictCategoryHandler) List(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	res, err := h.svc.List(c.Request.Context(), q)
	writeResult(c, res, err)
}

// Export GET /api/dict/categories/export
// @Summary      导出字典分类 CSV
// @Tags         dict-category
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /dict/categories/export [get]
func (h *DictCategoryHandler) Export(c *gin.Context) {
	csv, err := h.svc.Export(c.Request.Context())
	writeCSV(c, csv, err)
}

// Get GET /api/dict/categories/:id
// @Summary      字典分类详情
// @Tags         dict-category
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /dict/categories/{id} [get]
func (h *DictCategoryHandler) Get(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	e, err := h.svc.Get(c.Request.Context(), id)
	writeData(c, e, err)
}

// Create POST /api/dict/categories
// @Summary      创建字典分类
// @Tags         dict-category
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.DictCategoryUpsertReq true "分类信息"
// @Success      200  {object} response.ApiResult
// @Router       /dict/categories [post]
func (h *DictCategoryHandler) Create(c *gin.Context) {
	var req service.DictCategoryUpsertReq
	if !bindJSON(c, &req) {
		return
	}
	e := &model.DictCategory{Name: req.Name, Code: req.Code, Description: req.Description, Status: req.Status}
	if err := h.svc.Create(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "创建成功")
}

// Update PUT /api/dict/categories/:id
// @Summary      更新字典分类
// @Tags         dict-category
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "ID"
// @Param        request body service.DictCategoryUpsertReq true "分类信息"
// @Success      200  {object} response.ApiResult
// @Router       /dict/categories/{id} [put]
func (h *DictCategoryHandler) Update(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	var req service.DictCategoryUpsertReq
	if !bindJSON(c, &req) {
		return
	}
	e, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	e.Name, e.Code, e.Description, e.Status = req.Name, req.Code, req.Description, req.Status
	if err := h.svc.Update(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "更新成功")
}

// Delete DELETE /api/dict/categories/:id
// @Summary      删除字典分类
// @Tags         dict-category
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /dict/categories/{id} [delete]
func (h *DictCategoryHandler) Delete(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// BatchDelete DELETE /api/dict/categories
// @Summary      批量删除字典分类
// @Tags         dict-category
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dictBatchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Router       /dict/categories [delete]
func (h *DictCategoryHandler) BatchDelete(c *gin.Context) {
	var req dictBatchDeleteReq
	if !bindJSON(c, &req) {
		return
	}
	ids, ok := parseIDs(c, req.IDs)
	if !ok {
		return
	}
	if err := h.svc.BatchDelete(c.Request.Context(), ids); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// ==================== Level 2: 字典 ====================

// DictHandler 字典 HTTP 端点。
type DictHandler struct {
	svc *service.DictService
}

func NewDictHandler(svc *service.DictService) *DictHandler { return &DictHandler{svc: svc} }

// List GET /api/dict/dicts
// @Summary      字典分页列表
// @Tags         dict
// @Produce      json
// @Security     BearerAuth
// @Param        page       query int    false "页码" default(1)
// @Param        size       query int    false "每页" default(10)
// @Param        keyword    query string false "名称/编码关键词"
// @Param        categoryId query int    false "分类 ID"
// @Success      200  {object} response.ApiResult
// @Router       /dict/dicts [get]
func (h *DictHandler) List(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	res, err := h.svc.List(c.Request.Context(), q, uint(categoryID))
	writeResult(c, res, err)
}

// Export GET /api/dict/dicts/export
// @Summary      导出字典 CSV
// @Tags         dict
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /dict/dicts/export [get]
func (h *DictHandler) Export(c *gin.Context) {
	csv, err := h.svc.Export(c.Request.Context())
	writeCSV(c, csv, err)
}

// Get GET /api/dict/dicts/:id
// @Summary      字典详情
// @Tags         dict
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /dict/dicts/{id} [get]
func (h *DictHandler) Get(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	e, err := h.svc.Get(c.Request.Context(), id)
	writeData(c, e, err)
}

// Create POST /api/dict/dicts
// @Summary      创建字典
// @Tags         dict
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.DictUpsertReq true "字典信息"
// @Success      200  {object} response.ApiResult
// @Router       /dict/dicts [post]
func (h *DictHandler) Create(c *gin.Context) {
	var req service.DictUpsertReq
	if !bindJSON(c, &req) {
		return
	}
	e := &model.Dict{CategoryID: req.CategoryID, Name: req.Name, Code: req.Code, Description: req.Description, Status: req.Status}
	if err := h.svc.Create(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "创建成功")
}

// Update PUT /api/dict/dicts/:id
// @Summary      更新字典
// @Tags         dict
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "ID"
// @Param        request body service.DictUpsertReq true "字典信息"
// @Success      200  {object} response.ApiResult
// @Router       /dict/dicts/{id} [put]
func (h *DictHandler) Update(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	var req service.DictUpsertReq
	if !bindJSON(c, &req) {
		return
	}
	e, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	e.CategoryID, e.Name, e.Code, e.Description, e.Status = req.CategoryID, req.Name, req.Code, req.Description, req.Status
	if err := h.svc.Update(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "更新成功")
}

// Delete DELETE /api/dict/dicts/:id
// @Summary      删除字典
// @Tags         dict
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /dict/dicts/{id} [delete]
func (h *DictHandler) Delete(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// BatchDelete DELETE /api/dict/dicts
// @Summary      批量删除字典
// @Tags         dict
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dictBatchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Router       /dict/dicts [delete]
func (h *DictHandler) BatchDelete(c *gin.Context) {
	var req dictBatchDeleteReq
	if !bindJSON(c, &req) {
		return
	}
	ids, ok := parseIDs(c, req.IDs)
	if !ok {
		return
	}
	if err := h.svc.BatchDelete(c.Request.Context(), ids); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// ==================== Level 3: 字典项 ====================

// DictItemHandler 字典项 HTTP 端点。
type DictItemHandler struct {
	svc *service.DictItemService
}

func NewDictItemHandler(svc *service.DictItemService) *DictItemHandler {
	return &DictItemHandler{svc: svc}
}

// List GET /api/dict/items
// @Summary      字典项分页列表
// @Tags         dict-item
// @Produce      json
// @Security     BearerAuth
// @Param        page    query int    false "页码" default(1)
// @Param        size    query int    false "每页" default(10)
// @Param        keyword query string false "名称/编码关键词"
// @Param        dictId  query int    false "字典 ID"
// @Success      200  {object} response.ApiResult
// @Router       /dict/items [get]
func (h *DictItemHandler) List(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	dictID, _ := strconv.ParseUint(c.Query("dictId"), 10, 64)
	res, err := h.svc.List(c.Request.Context(), q, uint(dictID))
	writeResult(c, res, err)
}

// Export GET /api/dict/items/export
// @Summary      导出字典项 CSV
// @Tags         dict-item
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /dict/items/export [get]
func (h *DictItemHandler) Export(c *gin.Context) {
	csv, err := h.svc.Export(c.Request.Context())
	writeCSV(c, csv, err)
}

// Get GET /api/dict/items/:id
// @Summary      字典项详情
// @Tags         dict-item
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /dict/items/{id} [get]
func (h *DictItemHandler) Get(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	e, err := h.svc.Get(c.Request.Context(), id)
	writeData(c, e, err)
}

// Create POST /api/dict/items
// @Summary      创建字典项
// @Tags         dict-item
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.DictItemUpsertReq true "字典项信息"
// @Success      200  {object} response.ApiResult
// @Router       /dict/items [post]
func (h *DictItemHandler) Create(c *gin.Context) {
	var req service.DictItemUpsertReq
	if !bindJSON(c, &req) {
		return
	}
	e := &model.DictItem{DictID: req.DictID, Name: req.Name, Code: req.Code, Value: req.Value, Sort: req.Sort, Status: req.Status}
	if err := h.svc.Create(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "创建成功")
}

// Update PUT /api/dict/items/:id
// @Summary      更新字典项
// @Tags         dict-item
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "ID"
// @Param        request body service.DictItemUpsertReq true "字典项信息"
// @Success      200  {object} response.ApiResult
// @Router       /dict/items/{id} [put]
func (h *DictItemHandler) Update(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	var req service.DictItemUpsertReq
	if !bindJSON(c, &req) {
		return
	}
	e, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	e.DictID, e.Name, e.Code, e.Value, e.Sort, e.Status = req.DictID, req.Name, req.Code, req.Value, req.Sort, req.Status
	if err := h.svc.Update(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "更新成功")
}

// Delete DELETE /api/dict/items/:id
// @Summary      删除字典项
// @Tags         dict-item
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /dict/items/{id} [delete]
func (h *DictItemHandler) Delete(c *gin.Context) {
	id, ok := parseIDFromCtx(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// BatchDelete DELETE /api/dict/items
// @Summary      批量删除字典项
// @Tags         dict-item
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dictBatchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Router       /dict/items [delete]
func (h *DictItemHandler) BatchDelete(c *gin.Context) {
	var req dictBatchDeleteReq
	if !bindJSON(c, &req) {
		return
	}
	ids, ok := parseIDs(c, req.IDs)
	if !ok {
		return
	}
	if err := h.svc.BatchDelete(c.Request.Context(), ids); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// ===== handler 公共辅助（仅本文件三级用）=====

func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return false
	}
	return true
}

func writeResult(c *gin.Context, res any, err error) {
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

func writeData(c *gin.Context, data any, err error) {
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, data)
}

func writeCSV(c *gin.Context, csv string, err error) {
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, csv, "导出成功")
}
