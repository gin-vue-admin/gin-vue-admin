<!--
感谢你的贡献！请按下述 checklist 检查后再提交。
-->

## 变更说明

<!-- 简述本 PR 做了什么、为什么。关联 issue：Fixes #123 / Refs #456 -->

## 变更类型

- [ ] feat（新功能）
- [ ] fix（缺陷修复）
- [ ] refactor（重构）
- [ ] perf（性能优化）
- [ ] docs（文档/注释）
- [ ] test（测试）
- [ ] chore（构建/工具链）
- [ ] breaking change（破坏性变更）

## Checklist

- [ ] 分支从 `main` 切出，提交信息遵循 Conventional Commits
- [ ] `golangci-lint run ./...` 无告警
- [ ] `go test ./...` 全绿，新增功能带测试
- [ ] 改动 handler/路由/DTO 时，已更新 Swagger 注解并 `swag init` 重生成 docs 包
- [ ] 未引入新的直接依赖（如必须，已评估维护性与许可证）
- [ ] 不含密钥、密码、本地配置等敏感信息

## 影响范围与测试方式

<!-- 哪些模块受影响、如何验证、是否需要数据库迁移/seed 变更 -->
