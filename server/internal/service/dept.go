// Package service 部门业务：树形列表 + 过滤后重建树 + 递归级联删除 + CSV 导出。
// 演示基座对"树形业务"的支持：复用 GenericRepository 的 CRUD，树构建/级联删在 service 层。
package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/csvutil"
	"gva/internal/repository"

	"gorm.io/gorm"
)

// DeptInfo 部门树节点（对齐前端 DeptInfo 契约）。ParentID nil=根→JSON null。
type DeptInfo struct {
	ID         uint       `json:"id"`
	ParentID   *uint      `json:"parentId"`
	Name       string     `json:"name"`
	Leader     string     `json:"leader"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email"`
	Sort       int        `json:"sort"`
	Status     string     `json:"status"`
	Children   []DeptInfo `json:"children,omitempty"`
	CreateTime time.Time  `json:"createTime"`
	UpdateTime time.Time  `json:"updateTime"`
}

// DeptUpsertReq 创建/更新部门请求。ParentID nil=根。
type DeptUpsertReq struct {
	ParentID *uint  `json:"parentId"`
	Name     string `json:"name" binding:"required"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Sort     int    `json:"sort"`
	Status   string `json:"status" binding:"required,oneof=active inactive"`
}

// DeptService 部门业务。
type DeptService struct {
	repo repository.DeptRepository
}

// NewDeptService 构造部门服务。
func NewDeptService(repo repository.DeptRepository) *DeptService {
	return &DeptService{repo: repo}
}

// List 全量树形列表，按 keyword/status 过滤后重建树。
// 过滤产生的孤儿节点（父被过滤）提升为根，保证返回仍是合法森林。
func (s *DeptService) List(ctx context.Context, keyword, status string) ([]DeptInfo, error) {
	all, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return buildDeptTree(filterDepts(all, keyword, status)), nil
}

// Get 详情。不存在→404。
func (s *DeptService) Get(ctx context.Context, id uint) (*model.Dept, error) {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("部门不存在")
		}
		return nil, err
	}
	return d, nil
}

// Create 创建部门。
func (s *DeptService) Create(ctx context.Context, req *DeptUpsertReq) (*model.Dept, error) {
	d := &model.Dept{
		ParentID: deptParentID(req.ParentID), Name: req.Name, Leader: req.Leader,
		Phone: req.Phone, Email: req.Email, Sort: req.Sort, Status: req.Status,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// Update 全量更新。
func (s *DeptService) Update(ctx context.Context, d *model.Dept) error {
	return s.repo.Update(ctx, d)
}

// Delete 级联删除：自身 + 所有子孙（递归收集后代 id，批量软删）。不存在→404。
func (s *DeptService) Delete(ctx context.Context, id uint) error {
	all, err := s.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	if !deptExists(all, id) {
		return apperr.NotFound("部门不存在")
	}
	ids := []uint{id}
	collectDeptDescendants(all, id, &ids)
	return s.repo.BatchDelete(ctx, ids)
}

// BatchDelete 批量级联删除（每个节点 + 其子孙）。
func (s *DeptService) BatchDelete(ctx context.Context, ids []uint) error {
	all, err := s.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	var allIDs []uint
	for _, id := range ids {
		if !deptExists(all, id) {
			continue
		}
		allIDs = append(allIDs, id)
		collectDeptDescendants(all, id, &allIDs)
	}
	if len(allIDs) == 0 {
		return nil
	}
	return s.repo.BatchDelete(ctx, allIDs)
}

// Export 全量扁平 CSV（含表头），导出用。
func (s *DeptService) Export(ctx context.Context) (string, error) {
	all, err := s.repo.GetAll(ctx)
	if err != nil {
		return "", err
	}
	rows := make([]map[string]any, 0, len(all))
	for _, d := range all {
		rows = append(rows, map[string]any{
			"id":         strconv.FormatUint(uint64(d.ID), 10),
			"name":       d.Name,
			"leader":     d.Leader,
			"phone":      d.Phone,
			"email":      d.Email,
			"sort":       d.Sort,
			"status":     d.Status,
			"createTime": d.CreatedAt,
			"updateTime": d.UpdatedAt,
		})
	}
	return csvutil.Build(rows, []string{"id", "name", "leader", "phone", "email", "sort", "status", "createTime", "updateTime"}), nil
}

// filterDepts keyword 模糊匹配 name/leader，status 精确匹配。两者皆空返回原列表。
func filterDepts(all []model.Dept, keyword, status string) []model.Dept {
	if keyword == "" && status == "" {
		return all
	}
	out := make([]model.Dept, 0, len(all))
	for _, d := range all {
		if status != "" && d.Status != status {
			continue
		}
		if keyword != "" && !strings.Contains(d.Name, keyword) && !strings.Contains(d.Leader, keyword) {
			continue
		}
		out = append(out, d)
	}
	return out
}

// buildDeptTree 扁平部门→DeptInfo 树。
// 用递归深度优先构建：从根拷贝并递归填充子孙，避免多层值拷贝丢失深层子节点
// （menu 的三遍法仅适用 2 层；部门可能任意深度，故改递归）。
// ParentID=0 或父不在列表→根（过滤产生的孤儿提升为根）。
func buildDeptTree(depts []model.Dept) []DeptInfo {
	byID := make(map[uint]*DeptInfo, len(depts))
	for i := range depts {
		d := &depts[i]
		info := toDeptInfo(d)
		byID[d.ID] = &info
	}
	// 按 parent 收集子节点指针
	childrenOf := make(map[uint][]*DeptInfo)
	for i := range depts {
		d := &depts[i]
		if d.ParentID != 0 && byID[d.ParentID] != nil {
			childrenOf[d.ParentID] = append(childrenOf[d.ParentID], byID[d.ID])
		}
	}
	// 递归从节点 id 构建完整值子树（深度优先拷贝，子孙完整）
	var build func(id uint) DeptInfo
	build = func(id uint) DeptInfo {
		info := *byID[id]
		info.Children = nil
		for _, child := range childrenOf[id] {
			info.Children = append(info.Children, build(child.ID))
		}
		return info
	}
	roots := make([]DeptInfo, 0)
	for i := range depts {
		d := &depts[i]
		if d.ParentID == 0 || byID[d.ParentID] == nil {
			roots = append(roots, build(d.ID))
		}
	}
	return roots
}

func toDeptInfo(d *model.Dept) DeptInfo {
	var pid *uint
	if d.ParentID != 0 {
		p := d.ParentID
		pid = &p
	}
	return DeptInfo{
		ID: d.ID, ParentID: pid, Name: d.Name, Leader: d.Leader,
		Phone: d.Phone, Email: d.Email, Sort: d.Sort, Status: d.Status,
		CreateTime: d.CreatedAt, UpdateTime: d.UpdatedAt,
	}
}

func deptParentID(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

func deptExists(all []model.Dept, id uint) bool {
	for _, d := range all {
		if d.ID == id {
			return true
		}
	}
	return false
}

// collectDeptDescendants 递归收集 id 的所有后代主键。
func collectDeptDescendants(all []model.Dept, id uint, ids *[]uint) {
	for _, d := range all {
		if d.ParentID == id {
			*ids = append(*ids, d.ID)
			collectDeptDescendants(all, d.ID, ids)
		}
	}
}
