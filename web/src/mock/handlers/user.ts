import { http } from 'msw'
import { ok, fail, toCsv } from './_utils'

// 定义用户类型
interface UserInfo {
  id: string
  username: string
  realName: string
  email: string
  phone: string
  // 角色 code 集合（多角色），与 UserProfile.roles 对齐
  roles: string[]
  status: 'active' | 'inactive'
  avatar: string
  createTime: string
  lastLoginTime: string
  loginCount: number
}

// 生成随机用户数据
const generateUsers = (count: number): UserInfo[] => {
  const users: UserInfo[] = []
  const roleCodes = [
    'super_admin',
    'system_admin',
    'user_admin',
    'content_admin',
    'finance_admin',
    'data_analyst'
  ] as const
  const statuses = ['active', 'inactive'] as const
  const realNames = [
    '张三',
    '李四',
    '王五',
    '赵六',
    '钱七',
    '孙八',
    '周九',
    '吴十'
  ]

  for (let i = 1; i <= count; i++) {
    users.push({
      id: i.toString(),
      username: `user${i}`,
      realName: realNames[Math.floor(Math.random() * realNames.length)],
      email: `user${i}@example.com`,
      phone: `138${Math.floor(Math.random() * 100000000)
        .toString()
        .padStart(8, '0')}`,
      roles: [...roleCodes]
        .sort(() => Math.random() - 0.5)
        .slice(0, 1 + Math.floor(Math.random() * 2)),
      status: statuses[Math.floor(Math.random() * statuses.length)],
      avatar: `https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png`,
      createTime: new Date(
        Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000
      ).toISOString(),
      lastLoginTime: new Date(
        Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000
      ).toISOString(),
      loginCount: Math.floor(Math.random() * 1000)
    })
  }

  return users
}

const users: UserInfo[] = generateUsers(50)

export const userHandlers = [
  // 获取用户列表
  http.get('/api/user', ({ request }) => {
    const searchParams = new URL(request.url).searchParams
    const keyword = searchParams.get('keyword') || ''
    const role = searchParams.get('role') || ''
    const status = searchParams.get('status') || ''
    const page = parseInt(searchParams.get('page') || '1')
    const size = parseInt(searchParams.get('size') || '10')

    // 筛选
    const filteredUsers = users.filter((user) => {
      const matchKeyword =
        keyword === '' ||
        user.username.includes(keyword) ||
        user.realName.includes(keyword) ||
        user.email.includes(keyword) ||
        user.phone.includes(keyword)
      const matchRole = role === '' || user.roles.includes(role)
      const matchStatus = status === '' || user.status === status

      return matchKeyword && matchRole && matchStatus
    })

    // 分页
    const startIndex = (page - 1) * size
    const paginatedUsers = filteredUsers.slice(startIndex, startIndex + size)

    return ok({
      records: paginatedUsers,
      total: filteredUsers.length,
      current: page,
      size
    })
  }),

  // 导出用户列表，必须在 /:id 之前注册
  http.get('/api/user/export', () => {
    const csv = toCsv(
      users.map((u) => ({ ...u, roles: u.roles.join(';') })),
      [
        'username',
        'realName',
        'email',
        'phone',
        'roles',
        'status',
        'createTime',
        'lastLoginTime',
        'loginCount'
      ]
    )
    return ok(csv, '导出成功')
  }),

  // 获取用户详情
  http.get('/api/user/:id', ({ params }) => {
    const id = params.id as string
    const user = users.find((u) => u.id === id)

    if (user) {
      return ok(user)
    }

    return fail(404, '用户不存在')
  }),

  // 创建用户
  http.post('/api/user', async ({ request }) => {
    const body = (await request.json()) as Partial<UserInfo>
    const newUser: UserInfo = {
      id: (users.length + 1).toString(),
      username: body.username || '',
      realName: body.realName || '',
      email: body.email || '',
      phone: body.phone || '',
      roles: body.roles ?? [],
      status: body.status || 'active',
      avatar: `https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png`,
      createTime: new Date().toISOString(),
      lastLoginTime: new Date().toISOString(),
      loginCount: 0,
      ...body
    }

    users.push(newUser)

    return ok(newUser, '创建成功')
  }),

  // 更新用户
  http.put('/api/user/:id', async ({ request, params }) => {
    const id = params.id as string
    const userIndex = users.findIndex((u) => u.id === id)

    if (userIndex !== -1) {
      const body = (await request.json()) as Partial<UserInfo>
      users[userIndex] = {
        ...users[userIndex],
        ...body
      }

      return ok(users[userIndex], '更新成功')
    }

    return fail(404, '用户不存在')
  }),

  // 删除用户
  http.delete('/api/user/:id', ({ params }) => {
    const id = params.id as string
    const userIndex = users.findIndex((u) => u.id === id)

    if (userIndex !== -1) {
      users.splice(userIndex, 1)

      return ok(true, '删除成功')
    }

    return fail(404, '用户不存在')
  }),

  // 批量删除用户
  http.delete('/api/user', async ({ request }) => {
    const body = (await request.json()) as { ids: string[] }

    body.ids.forEach((id: string) => {
      const userIndex = users.findIndex((u) => u.id === id)
      if (userIndex !== -1) {
        users.splice(userIndex, 1)
      }
    })

    return ok(true, '删除成功')
  })
]
