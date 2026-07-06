// Package service dashboard 首页统计聚合（M-dashboard）。
// 用基座真实数据：users/roles/permissions/menus 计数 + operation_logs 最近活动与趋势。
// orders/revenue 字段为前端 demo 契约，基座无业务实体，映射为角色/权限计数（占位）。
package service

import (
	"context"
	"sort"
	"strconv"
	"time"

	"gva/internal/model"

	"gorm.io/gorm"
)

// StatItem 单项统计（对齐前端 dashboard/api.ts）。
type StatItem struct {
	Value    int64 `json:"value"`
	TrendPct int   `json:"trendPct"` // 环比百分比；基座无历史快照，固定 0
}

// StatsResp 首页统计卡片数据。
// orders/revenue 为前端 demo 字段，基座映射为角色数/权限数。
type StatsResp struct {
	Users   StatItem `json:"users"`
	Orders  StatItem `json:"orders"`  // → 角色数
	Revenue StatItem `json:"revenue"` // → 权限数
	Active  StatItem `json:"active"`  // → 菜单数
}

// ActivityResp 最近活动条目。
type ActivityResp struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Time  string `json:"time"`
	Type  string `json:"type"` // primary | success | warning | danger
}

// TrendPoint 趋势数据点。
type TrendPoint struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

// DistItem 分布项。
type DistItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// ChartsResp 图表数据。
type ChartsResp struct {
	Trend        []TrendPoint `json:"trend"`
	Distribution []DistItem   `json:"distribution"`
}

// DashboardService 首页统计服务，注入 gorm.DB 做聚合查询。
type DashboardService struct {
	db *gorm.DB
}

// NewDashboardService 构造首页统计服务。
func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

// Stats 返回 users/roles/permissions/menus 计数。
func (s *DashboardService) Stats(ctx context.Context) (*StatsResp, error) {
	var users, roles, perms, menus int64
	s.db.WithContext(ctx).Model(&model.User{}).Count(&users)
	s.db.WithContext(ctx).Model(&model.Role{}).Count(&roles)
	s.db.WithContext(ctx).Model(&model.Permission{}).Count(&perms)
	s.db.WithContext(ctx).Model(&model.Menu{}).Count(&menus)
	return &StatsResp{
		Users:   StatItem{Value: users},
		Orders:  StatItem{Value: roles},
		Revenue: StatItem{Value: perms},
		Active:  StatItem{Value: menus},
	}, nil
}

// Activities 返回最近 10 条操作日志（活动流）。
func (s *DashboardService) Activities(ctx context.Context) ([]ActivityResp, error) {
	var logs []model.OperationLog
	if err := s.db.WithContext(ctx).Order("created_at DESC").Limit(10).Find(&logs).Error; err != nil {
		return nil, err
	}
	res := make([]ActivityResp, 0, len(logs))
	for _, l := range logs {
		username := l.Username
		if username == "" {
			username = "匿名"
		}
		res = append(res, ActivityResp{
			ID:    strconv.FormatUint(uint64(l.ID), 10),
			Title: username + " " + l.Method,
			Desc:  l.Path,
			Time:  l.CreatedAt.Format(time.RFC3339),
			Type:  activityType(l.HTTPCode),
		})
	}
	return res, nil
}

// Charts 返回趋势（按天）与分布（按 method），range 控制天数。
func (s *DashboardService) Charts(ctx context.Context, days int) (*ChartsResp, error) {
	since := time.Now().AddDate(0, 0, -days)
	var logs []model.OperationLog
	if err := s.db.WithContext(ctx).Where("created_at >= ?", since).Find(&logs).Error; err != nil {
		return nil, err
	}
	trendMap := map[string]int64{}
	distMap := map[string]int64{}
	for _, l := range logs {
		trendMap[l.CreatedAt.Format("2006-01-02")]++
		distMap[l.Method]++
	}
	// 趋势：补齐每日序列（无数据日为 0）
	trend := make([]TrendPoint, 0, days)
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		trend = append(trend, TrendPoint{Date: d, Value: trendMap[d]})
	}
	// 分布：按 method 排序输出
	dist := make([]DistItem, 0, len(distMap))
	for name, v := range distMap {
		dist = append(dist, DistItem{Name: name, Value: v})
	}
	sort.Slice(dist, func(i, j int) bool { return dist[i].Value > dist[j].Value })
	return &ChartsResp{Trend: trend, Distribution: dist}, nil
}

// activityType 按 HTTP 状态码映射活动条目类型。
func activityType(code int) string {
	switch {
	case code >= 500:
		return "danger"
	case code >= 400:
		return "warning"
	case code >= 200 && code < 300:
		return "success"
	default:
		return "primary"
	}
}
