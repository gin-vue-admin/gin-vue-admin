// Package async 抽象"主路径之外"的后台执行，便于测试注入同步实现。
// 生产用 GoroutineRunner（go func）；测试用 SyncRunner（同步执行），消除 goroutine 与测试 teardown 的竞态。
// 后续 M5 可给 GoroutineRunner 加 Wait() 支持优雅停机。
package async

// Runner 后台任务执行抽象。
type Runner interface {
	Go(func())
}

// GoroutineRunner 生产实现：派发 goroutine。
type GoroutineRunner struct{}

// Go 派发一个后台 goroutine。
func (GoroutineRunner) Go(f func()) { go f() }

// SyncRunner 测试实现：同步执行（Login 返回前任务已完成，零竞态）。
type SyncRunner struct{}

// Go 同步执行任务。
func (SyncRunner) Go(f func()) { f() }
