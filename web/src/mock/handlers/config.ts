import { http } from 'msw'
import { ok, fail } from './_utils'

// 系统参数配置类型（对齐前端 SysConfigInfo / 后端 sys_config）
interface SysConfigInfo {
  id: string
  configKey: string
  configValue: string
  configName: string
  remark: string
  type: string
  createTime: string
  updateTime: string
}

// 种子数据（对齐后端 sys_config seed：site_title / default_page_size / login_captcha_enabled / token_expire_seconds）
const configs: SysConfigInfo[] = [
  {
    id: '1',
    configKey: 'site_title',
    configValue: 'gin-vue-admin 企业级后台管理系统',
    configName: '站点标题',
    remark: '浏览器标签页与登录页展示的系统名称',
    type: 'string',
    createTime: '2026-07-04T10:00:00.000Z',
    updateTime: '2026-07-04T10:00:00.000Z'
  },
  {
    id: '2',
    configKey: 'default_page_size',
    configValue: '10',
    configName: '默认分页大小',
    remark: '列表页默认每页条数',
    type: 'int',
    createTime: '2026-07-04T10:00:00.000Z',
    updateTime: '2026-07-04T10:00:00.000Z'
  },
  {
    id: '3',
    configKey: 'login_captcha_enabled',
    configValue: 'false',
    configName: '登录验证码开关',
    remark: '是否开启登录图形验证码',
    type: 'bool',
    createTime: '2026-07-04T10:00:00.000Z',
    updateTime: '2026-07-04T10:00:00.000Z'
  },
  {
    id: '4',
    configKey: 'token_expire_seconds',
    configValue: '7200',
    configName: 'Token 有效期（秒）',
    remark: 'JWT access token 过期时间',
    type: 'int',
    createTime: '2026-07-04T10:00:00.000Z',
    updateTime: '2026-07-04T10:00:00.000Z'
  }
]

export const configHandlers = [
  // 分页列表（keyword 模糊匹配 configKey / configName）
  http.get('/api/system/config', ({ request }) => {
    const searchParams = new URL(request.url).searchParams
    const keyword = searchParams.get('keyword') || ''
    const page = parseInt(searchParams.get('page') || '1')
    const size = parseInt(searchParams.get('size') || '10')

    const filtered = configs.filter(
      (c) =>
        keyword === '' ||
        c.configKey.includes(keyword) ||
        c.configName.includes(keyword)
    )

    const startIndex = (page - 1) * size
    const records = filtered.slice(startIndex, startIndex + size)

    return ok({
      records,
      total: filtered.length,
      current: page,
      size
    })
  }),

  // 按 key 查询（必须在 /:id 之前注册，否则 "key" 会被 :id 捕获）
  http.get('/api/system/config/key/:key', ({ params }) => {
    const key = params.key as string
    const config = configs.find((c) => c.configKey === key)
    return config ? ok(config) : fail(404, '配置不存在')
  }),

  // 按 id 查详情
  http.get('/api/system/config/:id', ({ params }) => {
    const id = params.id as string
    const config = configs.find((c) => c.id === id)
    return config ? ok(config) : fail(404, '配置不存在')
  }),

  // 创建（configKey 唯一校验，对齐后端）
  http.post('/api/system/config', async ({ request }) => {
    const body = (await request.json()) as Partial<SysConfigInfo>
    if (body.configKey && configs.some((c) => c.configKey === body.configKey)) {
      return fail(409, '配置键已存在')
    }
    const now = new Date().toISOString()
    const newConfig: SysConfigInfo = {
      id: (configs.length + 1).toString(),
      configKey: body.configKey || '',
      configValue: body.configValue || '',
      configName: body.configName || '',
      remark: body.remark || '',
      type: body.type || 'string',
      createTime: now,
      updateTime: now
    }
    configs.push(newConfig)
    return ok(newConfig, '创建成功')
  }),

  // 更新
  http.put('/api/system/config/:id', async ({ request, params }) => {
    const id = params.id as string
    const idx = configs.findIndex((c) => c.id === id)
    if (idx === -1) return fail(404, '配置不存在')
    const body = (await request.json()) as Partial<SysConfigInfo>
    configs[idx] = {
      ...configs[idx],
      ...body,
      updateTime: new Date().toISOString()
    }
    return ok(configs[idx], '更新成功')
  }),

  // 删除（sys_config 数据量小，按 YAGNI 不提供批量删，对齐前端 api.ts）
  http.delete('/api/system/config/:id', ({ params }) => {
    const id = params.id as string
    const idx = configs.findIndex((c) => c.id === id)
    if (idx === -1) return fail(404, '配置不存在')
    configs.splice(idx, 1)
    return ok(true, '删除成功')
  })
]
