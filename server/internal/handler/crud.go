// Package handler 通用 CRUD 示例端点（脚手架范例）。
// 展示新模块的最简 handler：绑定→调 service→写响应。对齐前端 /api/crud 契约。
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

// crudListQuery 列表查询参数。前端 crud demo 用 current（非 page）与 name（非 keyword），
// 此处映射到 pagination.Query{Page,Size,Keyword}。
type crudListQuery struct {
	Current int    `form:"current"`
	Size    int    `form:"size"`
	Name    string `form:"name"`
}

// crudUpsertReq 创建/更新共用请求体（对齐前端 crud item 契约）。
type crudUpsertReq struct {
	Date     string `json:"date"`
	Name     string `json:"name" binding:"required"`
	Province string `json:"province"`
	City     string `json:"city"`
	Address  string `json:"address"`
	Zip      int    `json:"zip"`
}

// crudBatchDeleteReq 批量删除请求体（前端契约 ids 为字符串数组）。
// 加 crud 前缀避免与同包其他 batchDeleteReq 冲突。
type crudBatchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// CrudHandler 通用 CRUD 端点处理器。
type CrudHandler struct {
	svc *service.CrudService
}

// NewCrudHandler 构造 crud 处理器。
func NewCrudHandler(svc *service.CrudService) *CrudHandler {
	return &CrudHandler{svc: svc}
}

// List GET /api/crud?current=1&size=10&name=xxx
// @Summary      CRUD 范例分页列表
// @Tags         crud
// @Produce      json
// @Security     BearerAuth
// @Param        current query int    false "页码" default(1)
// @Param        size    query int    false "每页" default(10)
// @Param        name    query string false "名称关键词"
// @Success      200  {object} response.ApiResult
// @Router       /crud [get]
func (h *CrudHandler) List(c *gin.Context) {
	var q crudListQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Current, Size: q.Size, Keyword: q.Name}
	res, err := h.svc.List(c.Request.Context(), pq)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Get GET /api/crud/:id
// @Summary      CRUD 范例详情
// @Tags         crud
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /crud/{id} [get]
func (h *CrudHandler) Get(c *gin.Context) {
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

// Create POST /api/crud
// @Summary      CRUD 范例创建
// @Tags         crud
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body crudUpsertReq true "条目"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Router       /crud [post]
func (h *CrudHandler) Create(c *gin.Context) {
	var req crudUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	e := &model.CrudItem{
		Date: req.Date, Name: req.Name, Province: req.Province,
		City: req.City, Address: req.Address, Zip: req.Zip,
	}
	if err := h.svc.Create(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "创建成功")
}

// Update PUT /api/crud/:id（先查再改字段再保存）
// @Summary      CRUD 范例更新
// @Tags         crud
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "ID"
// @Param        request body crudUpsertReq true "条目"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /crud/{id} [put]
func (h *CrudHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req crudUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	e, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	e.Date = req.Date
	e.Name = req.Name
	e.Province = req.Province
	e.City = req.City
	e.Address = req.Address
	e.Zip = req.Zip
	if err := h.svc.Update(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "更新成功")
}

// Delete DELETE /api/crud/:id
// @Summary      CRUD 范例删除
// @Tags         crud
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /crud/{id} [delete]
func (h *CrudHandler) Delete(c *gin.Context) {
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

// BatchDelete DELETE /api/crud （body: {ids:["1","2"]}）
// @Summary      CRUD 范例批量删除
// @Tags         crud
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body crudBatchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Router       /crud [delete]
func (h *CrudHandler) BatchDelete(c *gin.Context) {
	var req crudBatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	ids := make([]uint, 0, len(req.IDs))
	for _, s := range req.IDs {
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			apperr.Write(c, apperr.Validation("无效的 id: "+s, nil))
			return
		}
		ids = append(ids, uint(id))
	}
	if err := h.svc.BatchDelete(c.Request.Context(), ids); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}
