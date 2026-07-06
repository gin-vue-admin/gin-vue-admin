// system/config 领域 API（M10 系统参数配置）。
// 走 @/lib/http/client 的 api 辅助函数（已解包 data）。
import { api } from '@/lib/http/client'

// 系统配置类型
export interface SysConfigInfo {
  id: string
  configKey: string
  configValue: string
  configName: string
  remark: string
  type: string
  createTime: string
  updateTime: string
}

// 搜索参数
export interface SysConfigSearchRequest {
  keyword: string
  page: number
  size: number
}

// 创建/更新参数（对齐后端 sysConfigUpsertReq）
export interface SysConfigUpsertRequest {
  configKey: string
  configValue: string
  configName?: string
  remark?: string
  type?: string
}

// 列表响应
export interface SysConfigSearchResponse {
  records: SysConfigInfo[]
  total: number
  current: number
  size: number
}

// 分页列表
export const fetchSysConfigList = (params: SysConfigSearchRequest) =>
  api.get<SysConfigSearchResponse>('/api/system/config', { params })

// 详情（按 ID）
export const fetchSysConfigDetail = (id: string) =>
  api.get<SysConfigInfo>(`/api/system/config/${id}`)

// 详情（按 key）
export const fetchSysConfigByKey = (key: string) =>
  api.get<SysConfigInfo>(`/api/system/config/key/${key}`)

// 创建
export const createSysConfig = (data: SysConfigUpsertRequest) =>
  api.post<SysConfigInfo>('/api/system/config', data)

// 更新
export const updateSysConfig = (id: string, data: SysConfigUpsertRequest) =>
  api.put<SysConfigInfo>(`/api/system/config/${id}`, data)

// 删除（sys_config 数据量小，按 YAGNI 不提供批量删）
export const deleteSysConfig = (id: string) =>
  api.del<boolean>(`/api/system/config/${id}`)
