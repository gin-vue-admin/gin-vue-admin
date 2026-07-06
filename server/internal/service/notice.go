// Package service notice 公告业务（CRUD + 发布/撤销 + 导出）。
package service

import (
	"context"
	"strconv"
	"time"

	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/csvutil"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"
)

// NoticeUpsertReq 公告创建/更新请求。
type NoticeUpsertReq struct {
	Title       string `json:"title" binding:"required,max=255"`
	Content     string `json:"content"`
	Type        string `json:"type" binding:"omitempty,oneof=announcement notice todo"`
	Status      string `json:"status" binding:"omitempty,oneof=published draft expired"`
	Priority    string `json:"priority" binding:"omitempty,oneof=high medium low"`
	PublishTime string `json:"publishTime"`
	ExpireTime  string `json:"expireTime"`
}

// NoticeService 公告业务服务。
type NoticeService struct {
	repo     repository.NoticeRepository
	userRepo repository.UserRepository // 可选：Create 时反查 publisher 用户名
}

// NewNoticeService 构造公告服务。userRepo 用于反查发布人用户名（可为 nil）。
func NewNoticeService(repo repository.NoticeRepository, userRepo repository.UserRepository) *NoticeService {
	return &NoticeService{repo: repo, userRepo: userRepo}
}

// currentUsername 从 audit ctx 的 userID 反查用户名（公告发布人）。
// userRepo 缺失或查不到返回空串（不影响主流程）。
func (s *NoticeService) currentUsername(ctx context.Context) string {
	if s.userRepo == nil {
		return ""
	}
	uid, ok := audit.UserIDFrom(ctx)
	if !ok || uid == 0 {
		return ""
	}
	u, err := s.userRepo.FindByID(ctx, uid)
	if err != nil {
		return ""
	}
	return u.Username
}

func (s *NoticeService) List(ctx context.Context, q pagination.Query, noticeType string) (pagination.Result[model.Notice], error) {
	q.Normalize()
	return s.repo.List(ctx, q, noticeType)
}

func (s *NoticeService) Get(ctx context.Context, id uint) (*model.Notice, error) {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("公告不存在")
	}
	return n, nil
}

func (s *NoticeService) Create(ctx context.Context, req *NoticeUpsertReq) (*model.Notice, error) {
	n := &model.Notice{
		Title:     req.Title,
		Content:   req.Content,
		Type:      defaultStr(req.Type, "notice"),
		Status:    defaultStr(req.Status, "draft"),
		Priority:  defaultStr(req.Priority, "medium"),
		Publisher: s.currentUsername(ctx),
	}
	applyTimes(n, req.PublishTime, req.ExpireTime)
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *NoticeService) Update(ctx context.Context, id uint, req *NoticeUpsertReq) (*model.Notice, error) {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("公告不存在")
	}
	n.Title = req.Title
	n.Content = req.Content
	if req.Type != "" {
		n.Type = req.Type
	}
	if req.Status != "" {
		n.Status = req.Status
	}
	if req.Priority != "" {
		n.Priority = req.Priority
	}
	applyTimes(n, req.PublishTime, req.ExpireTime)
	if err := s.repo.Update(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *NoticeService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.NotFound("公告不存在")
	}
	return nil
}

func (s *NoticeService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.repo.BatchDelete(ctx, ids)
}

// Publish 发布公告：status=published + 记录发布时间。
func (s *NoticeService) Publish(ctx context.Context, id uint) error {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperr.NotFound("公告不存在")
	}
	n.Status = "published"
	now := time.Now()
	n.PublishTime = &now
	return s.repo.Update(ctx, n)
}

// Revoke 撤销发布：status=draft。
func (s *NoticeService) Revoke(ctx context.Context, id uint) error {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperr.NotFound("公告不存在")
	}
	n.Status = "draft"
	return s.repo.Update(ctx, n)
}

// Export 全量导出 CSV。
func (s *NoticeService) Export(ctx context.Context) (string, error) {
	q := pagination.Query{Page: 1, Size: 10000}
	q.Normalize()
	res, err := s.repo.List(ctx, q, "")
	if err != nil {
		return "", err
	}
	headers := []string{"ID", "标题", "类型", "状态", "优先级", "发布者", "发布时间"}
	rows := make([]map[string]any, 0, len(res.Records))
	for _, n := range res.Records {
		pt := ""
		if n.PublishTime != nil {
			pt = n.PublishTime.Format("2006-01-02 15:04:05")
		}
		rows = append(rows, map[string]any{
			"ID": strconv.FormatUint(uint64(n.ID), 10), "标题": n.Title, "类型": n.Type,
			"状态": n.Status, "优先级": n.Priority, "发布者": n.Publisher, "发布时间": pt,
		})
	}
	return csvutil.Build(rows, headers), nil
}

// defaultStr 空字符串兜底默认值。
func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// applyTimes 解析 ISO 时间字符串到 *time.Time（空串则置 nil 清除）。
func applyTimes(n *model.Notice, publish, expire string) {
	n.PublishTime = parseTimePtr(publish)
	n.ExpireTime = parseTimePtr(expire)
}

func parseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
