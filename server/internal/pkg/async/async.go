// Package async 抽象"主路径之外"的后台执行，便于测试注入同步实现。
// 生产用 GoroutineRunner（go func）；测试用 SyncRunner（同步执行），消除 goroutine 与测试 teardown 的竞态。
// 后续 M5 可给 GoroutineRunner 加 Wait() 支持优雅停机。
package async

import (
	"runtime/debug"

	"gva/internal/pkg/log"
)

// Runner 后台任务执行抽象。
type Runner interface {
	Go(func())
}

// GoroutineRunner 生产实现：派发 goroutine。
type GoroutineRunner struct{}

// Go 派发一个后台 goroutine。
// recover 防护：异步任务 panic 不得拖垮整个进程；记录堆栈后吞掉。
func (GoroutineRunner) Go(f func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.S.Errorw("异步任务 panic 已恢复",
					"recover", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		f()
	}()
}

// SyncRunner 测试实现：同步执行（Login 返回前任务已完成，零竞态）。
type SyncRunner struct{}

// Go 同步执行任务。
// 测试场景故意不加 recover：panic 应冒泡让测试失败（fail-fast），而非被吞。
func (SyncRunner) Go(f func()) { f() }
