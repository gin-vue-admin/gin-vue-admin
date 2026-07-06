// cmd/scaffold 代码生成器：基于泛型 Repository 基类 + crud 范例模式，
// 输入实体名+字段，一键生成 model/repository/service/handler 4 文件。
//
// 用法：
//
//	go run ./cmd/scaffold -name Post -fields "title:string,content:text,views:int,status:string"
//
// 生成后按提示手动装配（AutoMigrate / main.go / router.go / seed）。详见 server/docs/new-module-guide.md。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Field 描述实体字段。
type Field struct {
	Name    string // Go 字段名（PascalCase，如 Title）
	Lower   string // json/查询名（title）
	GoType  string // Go 类型
	GormTag string // gorm tag
}

// Entity 描述待生成实体。
type Entity struct {
	Name   string // PascalCase（Post）
	Lower  string // 小写单数（post）
	Table  string // 表名（posts）
	Fields []Field
	// 模板用：第一个 string 字段（作 List keyword 模糊匹配默认）
	FirstStrField string
}

func main() {
	name := flag.String("name", "", "实体名 PascalCase（如 Post）")
	fieldsStr := flag.String("fields", "", `字段 "name:type,..."（string/text/int/int64/float64/bool/time）`)
	table := flag.String("table", "", "表名（默认 实体小写+s）")
	out := flag.String("out", ".", "server 根目录")
	flag.Parse()
	if *name == "" || *fieldsStr == "" {
		fmt.Fprintln(os.Stderr, `用法: scaffold -name Post -fields "title:string,content:text,views:int"`)
		os.Exit(1)
	}
	e, err := buildEntity(*name, *fieldsStr, *table)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	if err := gen(*out, e); err != nil {
		fmt.Fprintln(os.Stderr, "生成失败:", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 已生成 %s（表 %s）4 文件。\n\n", e.Name, e.Table)
	fmt.Println("手动装配步骤（详见 docs/new-module-guide.md）：")
	fmt.Printf("  1. internal/model/rbac.go AutoMigrate 追加: &%s{}\n", e.Name)
	fmt.Printf("  2. cmd/api/main.go 装配 %sRepo/Svc/Handler 并传 NewRouter\n", e.Lower)
	fmt.Printf("  3. internal/server/router.go 注册 /api/%s 路由组 + %s:list/create/edit/delete 权限码\n", e.Lower, e.Lower)
	fmt.Printf("  4. internal/service/auth.go seedPermissionCodes + seedMenus 加 %s 菜单\n", e.Lower)
}

func buildEntity(name, fieldsStr, table string) (Entity, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Entity{}, fmt.Errorf("name 不能为空")
	}
	e := Entity{Name: pascal(name), Lower: lower(name)}
	if table == "" {
		e.Table = e.Lower + "s"
	} else {
		e.Table = table
	}
	for _, part := range strings.Split(fieldsStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		fn := strings.TrimSpace(kv[0])
		ft := "string"
		if len(kv) == 2 {
			ft = strings.TrimSpace(kv[1])
		}
		gt, gorm := typeMap(ft)
		f := Field{Name: pascal(fn), Lower: lower(fn), GoType: gt, GormTag: gorm}
		if e.FirstStrField == "" && gt == "string" {
			e.FirstStrField = f.Name
		}
		e.Fields = append(e.Fields, f)
	}
	if e.FirstStrField == "" {
		e.FirstStrField = "ID" // 无 string 字段时 keyword 不生效（filter 仅对 string）
	}
	return e, nil
}

// typeMap 类型 → (Go 类型, gorm tag)。
func typeMap(t string) (string, string) {
	switch t {
	case "string":
		return "string", "size:255"
	case "text":
		return "string", "type:text"
	case "int":
		return "int", "default:0"
	case "int64":
		return "int64", "default:0"
	case "float64":
		return "float64", "default:0"
	case "bool":
		return "bool", "default:false"
	default:
		return "string", "size:255"
	}
}

func pascal(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
func lower(s string) string { return strings.ToLower(s) }

func gen(root string, e Entity) error {
	type item struct {
		rel  string
		tmpl string
	}
	items := []item{
		{"internal/model/%s.go", modelTmpl},
		{"internal/repository/%s.go", repoTmpl},
		{"internal/service/%s.go", serviceTmpl},
		{"internal/handler/%s.go", handlerTmpl},
	}
	for _, it := range items {
		path := filepath.Join(root, fmt.Sprintf(it.rel, e.Lower))
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("⚠️  跳过（已存在）: %s\n", path)
			continue
		}
		t, err := template.New("f").Parse(it.tmpl)
		if err != nil {
			return err
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := t.Execute(f, e); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		fmt.Printf("✓ %s\n", path)
	}
	return nil
}

// ===== 模板 =====

var modelTmpl = `package model

// {{.Name}} 由 scaffold 生成。
type {{.Name}} struct {
	Model
{{range .Fields}}	{{.Name}} {{.GoType}} ` + "`gorm:\"{{.GormTag}}\" json:\"{{.Lower}}\"`" + `
{{end}}}

func ({{.Name}}) TableName() string { return "{{.Table}}" }
`

var repoTmpl = `package repository

import (
	"context"

	"gorm.io/gorm"
	"gva/internal/model"
	"gva/internal/pkg/pagination"
)

// {{.Name}}Repository 嵌入 GenericRepository 复用 CRUD，仅重写 List 注入过滤。
type {{.Name}}Repository interface {
	List(ctx context.Context, q pagination.Query) (pagination.Result[model.{{.Name}}], error)
	FindByID(ctx context.Context, id uint) (*model.{{.Name}}, error)
	Create(ctx context.Context, e *model.{{.Name}}) error
	Update(ctx context.Context, e *model.{{.Name}}) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
}

type {{.Lower}}Repository struct {
	*GenericRepository[model.{{.Name}}]
}

func New{{.Name}}Repository(db *gorm.DB) {{.Name}}Repository {
	return &{{.Lower}}Repository{GenericRepository: NewGenericRepository[model.{{.Name}}](db)}
}

func (r *{{.Lower}}Repository) List(ctx context.Context, q pagination.Query) (pagination.Result[model.{{.Name}}], error) {
	return r.GenericRepository.List(ctx, q, func(db *gorm.DB) *gorm.DB {
		if q.Keyword != "" {
			return db.Where("{{.FirstStrField}} LIKE ?", "%"+q.Keyword+"%")
		}
		return db
	})
}
`

var serviceTmpl = `package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/csvutil"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"
)

type {{.Name}}Service struct{ repo repository.{{.Name}}Repository }

func New{{.Name}}Service(repo repository.{{.Name}}Repository) *{{.Name}}Service {
	return &{{.Name}}Service{repo: repo}
}

func (s *{{.Name}}Service) List(ctx context.Context, q pagination.Query) (pagination.Result[model.{{.Name}}], error) {
	q.Normalize()
	res, err := s.repo.List(ctx, q)
	if err != nil {
		return pagination.Result[model.{{.Name}}]{}, err
	}
	res.Current, res.Size = q.Page, q.Size
	return res, nil
}

func (s *{{.Name}}Service) Get(ctx context.Context, id uint) (*model.{{.Name}}, error) {
	e, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("{{.Lower}} 不存在")
		}
		return nil, err
	}
	return e, nil
}

func (s *{{.Name}}Service) Create(ctx context.Context, e *model.{{.Name}}) error { return s.repo.Create(ctx, e) }
func (s *{{.Name}}Service) Update(ctx context.Context, e *model.{{.Name}}) error { return s.repo.Update(ctx, e) }

func (s *{{.Name}}Service) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("{{.Lower}} 不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *{{.Name}}Service) BatchDelete(ctx context.Context, ids []uint) error { return s.repo.BatchDelete(ctx, ids) }

// Export 取全量生成 CSV（绕 Normalize 100 上限）。
func (s *{{.Name}}Service) Export(ctx context.Context) (string, error) {
	res, err := s.repo.List(ctx, pagination.Query{Page: 1, Size: 100000})
	if err != nil {
		return "", err
	}
	rows := make([]map[string]any, 0, len(res.Records))
	for _, r := range res.Records {
		rows = append(rows, map[string]any{
			"id":         strconv.FormatUint(uint64(r.ID), 10),
			"createTime": r.CreatedAt,
			"updateTime": r.UpdatedAt,
		})
	}
	_ = time.Now
	return csvutil.Build(rows, []string{"id", "createTime", "updateTime"}), nil
}
`

var handlerTmpl = `package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/pkg/response"
	"gva/internal/service"
)

type {{.Lower}}UpsertReq struct {
{{range .Fields}}	{{.Name}} {{.GoType}} ` + "`json:\"{{.Lower}}\"`" + `
{{end}}}

type {{.Lower}}BatchDeleteReq struct {
	IDs []string ` + "`json:\"ids\" binding:\"required,min=1\"`" + `
}

type {{.Name}}Handler struct{ svc *service.{{.Name}}Service }

func New{{.Name}}Handler(svc *service.{{.Name}}Service) *{{.Name}}Handler {
	return &{{.Name}}Handler{svc: svc}
}

func (h *{{.Name}}Handler) List(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	res, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

func (h *{{.Name}}Handler) Get(c *gin.Context) {
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

func (h *{{.Name}}Handler) Create(c *gin.Context) {
	var req {{.Lower}}UpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	e := &model.{{.Name}}{
{{range .Fields}}		{{.Name}}: req.{{.Name}},
{{end}}	}
	if err := h.svc.Create(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "创建成功")
}

func (h *{{.Name}}Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req {{.Lower}}UpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	e, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
{{range .Fields}}	e.{{.Name}} = req.{{.Name}}
{{end}}	if err := h.svc.Update(c.Request.Context(), e); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e, "更新成功")
}

func (h *{{.Name}}Handler) Delete(c *gin.Context) {
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

func (h *{{.Name}}Handler) BatchDelete(c *gin.Context) {
	var req {{.Lower}}BatchDeleteReq
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
`
