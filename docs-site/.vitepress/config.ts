import { defineConfig } from 'vitepress'

// VitePress 配置 — 见 https://vitepress.dev/reference/site-config
// base 用于云主机 nginx 子路径部署（/docs/）；本地 dev 与 GitHub Pages 可经 DOCS_BASE 覆盖。
// cleanUrls=false：生成 /guide/xxx.html，nginx 无需 SPA fallback 即可直出静态。
export default defineConfig({
  title: 'gin-vue-admin',
  description: '业界优秀的通用管理框架基座 · Gin + Vue3 + Clean Architecture',
  lang: 'zh-CN',
  lastUpdated: true,
  cleanUrls: false,
  // dev 模式默认 base=/（直接访问根，便于本地预览）；
  // 生产 build 经 DOCS_BASE 覆盖为子路径（如 /docs/），见 Makefile docs-build / web.Dockerfile。
  base: process.env.DOCS_BASE ?? '/',
  // 仅忽略 localhost 示例链接（开发期 swagger 等本地地址），其他内部/外链死链严格报错。
  ignoreDeadLinks: 'localhostLinks',
  head: [['meta', { name: 'theme-color', content: '#00ADD8' }]],
  themeConfig: {
    logo: '/logo.png',
    search: { provider: 'local' },
    nav: [
      { text: '首页', link: '/' },
      { text: '指南', link: '/guide/overview' },
      { text: '架构', link: '/guide/architecture' },
      { text: 'API', link: '/guide/api' },
      { text: '部署', link: '/guide/deployment' },
      { text: 'GitHub', link: 'https://github.com/gin-vue-admin/gin-vue-admin' }
    ],
    sidebar: {
      '/guide/': [
        {
          text: '入门',
          items: [
            { text: '概览', link: '/guide/overview' },
            { text: '快速开始', link: '/guide/quickstart' },
            { text: '架构', link: '/guide/architecture' }
          ]
        },
        {
          text: '开发',
          items: [
            { text: '新增业务模块', link: '/guide/new-module' },
            { text: 'API 契约', link: '/guide/api' },
            { text: '差异化能力', link: '/guide/features' },
            { text: '测试', link: '/guide/testing' }
          ]
        },
        {
          text: '运维',
          items: [
            { text: '部署', link: '/guide/deployment' },
            { text: '安全', link: '/guide/security' },
            { text: '贡献指南', link: '/guide/contributing' }
          ]
        }
      ]
    },
    socialLinks: [{ icon: 'github', link: 'https://github.com/gin-vue-admin/gin-vue-admin' }],
    footer: {
      message: '基于 MIT 协议发布',
      copyright: 'Copyright © 2026 gin-vue-admin contributors'
    }
  }
})
