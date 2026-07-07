// Package handler 部门管理端点：树形列表 + CRUD + 级联删除 + 导出。
package handler

import (
	"strconv"

	"gva/internal/pkg/apperr"
	"gva/internal/pkg/response"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
)

// deptListQuery 列表查询参数（树形不分页，仅 keyword/status 过滤）。
type deptListQuery struct {
	Keyword string `form:"keyword"`
	Status  string `form:"status"`
}

// deptBatchDeleteReq 批量删除请求体（前端契约 ids 为字符串数组）。
type deptBatchDeleteReq = batchDeleteReq

// DeptHandler 部门端点处理器。
type DeptHandler struct {
	svc *service.DeptService
}

// NewDeptHandler 构造部门处理器。
func NewDeptHandler(svc *service.DeptService) *DeptHandler {
	return &DeptHandler{svc: svc}
}

// List GET /api/system/dept?keyword=&status=  返回部门树。
// @Summary      部门树
// @Tags         dept
// @Produce      json
// @Security     BearerAuth
// @Param        keyword query string false "名称关键词"
// @Param        status  query string false "状态" Enums(active, inactive)
// @Success      200  {object} response.ApiResult
// @Router       /system/dept [get]
func (h *DeptHandler) List(c *gin.Context) {
	var q deptListQuery
	_ = c.ShouldBindQuery(&q)
	tree, err := h.svc.List(c.Request.Context(), q.Keyword, q.Status)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, tree)
}

// Export GET /api/system/dept/export  返回 CSV 文本。
// @Summary      导出部门 CSV
// @Tags         dept
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /system/dept/export [get]
func (h *DeptHandler) Export(c *gin.Context) {
	csv, err := h.svc.Export(c.Request.Context())
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, csv, "导出成功")
}

// Get GET /api/system/dept/:id
// @Summary      部门详情
// @Tags         dept
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "部门 ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/dept/{id} [get]
func (h *DeptHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	d, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, d)
}

// Create POST /api/system/dept
// @Summary      创建部门
// @Tags         dept
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.DeptUpsertReq true "部门信息"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Router       /system/dept [post]
func (h *DeptHandler) Create(c *gin.Context) {
	var req service.DeptUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	d, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, d, "创建成功")
}

// Update PUT /api/system/dept/:id（先查再改字段再保存）
// @Summary      更新部门
// @Tags         dept
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "部门 ID"
// @Param        request body service.DeptUpsertReq true "部门信息"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/dept/{id} [put]
func (h *DeptHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req service.DeptUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	d, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	d.ParentID = deptParentIDFromReq(req.ParentID)
	d.Name = req.Name
	d.Leader = req.Leader
	d.Phone = req.Phone
	d.Email = req.Email
	d.Sort = req.Sort
	d.Status = req.Status
	if err := h.svc.Update(c.Request.Context(), d); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, d, "更新成功")
}

// Delete DELETE /api/system/dept/:id（级联删除子孙）
// @Summary      删除部门（级联子孙）
// @Tags         dept
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "部门 ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Failure      409  {object} response.ProblemDetail "存在子部门或关联用户"
// @Router       /system/dept/{id} [delete]
func (h *DeptHandler) Delete(c *gin.Context) {
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

// BatchDelete DELETE /api/system/dept（body: {ids:["1","2"]}，每个节点级联删子孙）
// @Summary      批量删除部门
// @Tags         dept
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body deptBatchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Router       /system/dept [delete]
func (h *DeptHandler) BatchDelete(c *gin.Context) {
	var req deptBatchDeleteReq
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

// deptParentIDFromReq *uint→uint（nil→0=根）。
func deptParentIDFromReq(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}
