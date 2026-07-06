---
layout: home

hero:
  name: gin-vue-admin
  text: 通用管理框架基座
  tagline: Gin + Vue3 + Clean Architecture · 对标 RuoYi/go-admin/django-admin
  image:
    src: /logo.png
    alt: gin-vue-admin
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/quickstart
    - theme: alt
      text: 架构总览
      link: /guide/architecture
    - theme: alt
      text: GitHub
      link: https://github.com/gin-vue-admin/gin-vue-admin

features:
  - icon: 🏗️
    title: Clean Architecture
    details: handler → service → repository → model 单向依赖，泛型 Repository 基类，构造注入。
  - icon: 🔐
    title: 完整 RBAC
    details: 用户/角色/权限/菜单 + JWT 双 token + 防枚举登录 + 权限缓存 TTL。
  - icon: 📊
    title: 数据范围
    details: 按角色 DataScope（all/dept/dept_and_sub/self）控制可见数据，超管短路。
  - icon: 📝
    title: 全栈审计
    details: 操作日志中间件 + 登录日志 + 审计字段（CreatedBy/UpdatedBy/DeletedBy）。
  - icon: 🎛️
    title: 运营配置
    details: sys_config + 内存缓存 + 编程 API，运营改配置不发版。
  - icon: 🧩
    title: 代码生成器
    details: CLI 一键生成 model/repo/service/handler 四层代码，复用泛型基类。
  - icon: 📚
    title: Swagger 文档
    details: 84 个端点全注解，访问 /swagger/index.html 在线调试。
  - icon: 🚀
    title: 一键部署
    details: Docker Compose 全栈编排 + nginx 反代 + Let's Encrypt TLS。
---
