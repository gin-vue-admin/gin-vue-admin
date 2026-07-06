<template>
  <SearchTable
    title="用户管理"
    :loading="loading"
    :data="tableData"
    :columns="columns"
    :pagination="pagination"
    :selected-rows="tableSelectedRows"
    selectable
    row-key="id"
    @search="handleSearch"
    @reset="handleReset"
    @page-change="handlePageChange"
    @selection-change="onSelectionChange"
  >
    <template #search>
      <el-input
        v-model="searchForm.keyword"
        placeholder="用户名、姓名、邮箱或电话"
        clearable
        style="width: 220px"
        @keyup.enter="handleSearch"
      />
      <el-select
        v-model="searchForm.role"
        clearable
        placeholder="角色"
        style="width: 160px"
      >
        <el-option
          v-for="r in roleOptions"
          :key="r.code"
          :label="r.name"
          :value="r.code"
        />
      </el-select>
      <el-select
        v-model="searchForm.status"
        clearable
        placeholder="状态"
        style="width: 120px"
      >
        <el-option
          label="启用"
          value="active"
        />
        <el-option
          label="禁用"
          value="inactive"
        />
      </el-select>
    </template>

    <template #actions>
      <el-button
        type="primary"
        :icon="Plus"
        @click="openDrawer('add')"
      >
        新增用户
      </el-button>
      <el-button
        type="danger"
        :icon="Delete"
        :disabled="selectedRows.length === 0"
        @click="handleBatchDelete"
      >
        批量删除
      </el-button>
      <el-button
        :icon="Refresh"
        @click="fetchList"
      >
        刷新
      </el-button>
    </template>

    <template #col-roles="{ row }">
      <el-tag
        v-for="code in row.roles"
        :key="code"
        type="info"
        size="small"
        style="margin: 2px"
      >
        {{ roleMap[code] ?? code }}
      </el-tag>
    </template>

    <template #col-status="{ row }">
      <StatusTag :status="row.status" />
    </template>

    <template #col-createTime="{ row }">
      {{ formatDate(row.createTime) }}
    </template>

    <template #col-lastLoginTime="{ row }">
      {{ formatDate(row.lastLoginTime) }}
    </template>

    <template #col-actions="{ row }">
      <el-button
        link
        type="primary"
        size="small"
        @click="openDrawer('view', row)"
      >
        查看
      </el-button>
      <el-button
        link
        type="primary"
        size="small"
        @click="openDrawer('edit', row)"
      >
        编辑
      </el-button>
      <el-button
        link
        type="danger"
        size="small"
        @click="handleDelete(row.id)"
      >
        删除
      </el-button>
    </template>
  </SearchTable>

  <UserFormDrawer
    v-model="drawerVisible"
    :mode="drawerMode"
    :data="editingUser"
    @success="onFormSuccess"
  />
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { Plus, Delete, Refresh } from '@element-plus/icons-vue'
import { SearchTable, StatusTag } from '@/app/components'
import { useCrud } from '@/app/composables/useCrud'
import { formatDate } from '@/lib/format'
import type { ColumnDef } from '@/app/components/SearchTable/types'
import {
  fetchUserList,
  deleteUser,
  batchDeleteUsers,
  type UserInfo,
  type UserSearchRequest,
} from '../api'
import { fetchRoleList, type RoleInfo } from '@/modules/system/role/api'
import UserFormDrawer from './UserFormDrawer.vue'

const {
  listData,
  loading,
  pagination,
  searchForm,
  selectedRows,
  fetchList,
  handleSearch,
  handleReset,
  handlePageChange,
  handleSelectionChange,
  handleDelete,
  handleBatchDelete,
} = useCrud<UserInfo>({
  fetch: (params) => fetchUserList(params as unknown as UserSearchRequest),
  remove: deleteUser,
  batchRemove: batchDeleteUsers,
  defaultSearchForm: { keyword: '', role: '', status: '' },
  pageSize: 10,
})

const onSelectionChange = (rows: Record<string, unknown>[]) => {
  handleSelectionChange(rows as unknown as UserInfo[])
}

const tableData = computed(() => listData.value as unknown as Record<string, unknown>[])
const tableSelectedRows = computed(() => selectedRows.value as unknown as Record<string, unknown>[])

const columns: ColumnDef[] = [
  { prop: 'username', label: '用户名', minWidth: 120 },
  { prop: 'realName', label: '姓名', minWidth: 100 },
  { prop: 'email', label: '邮箱', minWidth: 180 },
  { prop: 'phone', label: '电话', minWidth: 130 },
  { prop: 'roles', label: '角色', minWidth: 140, slot: 'roles' },
  { prop: 'status', label: '状态', minWidth: 90, slot: 'status' },
  { prop: 'createTime', label: '创建时间', minWidth: 170, slot: 'createTime' },
  { prop: 'lastLoginTime', label: '最后登录', minWidth: 170, slot: 'lastLoginTime' },
  { prop: 'actions', label: '操作', width: 200, fixed: 'right', slot: 'actions' },
]

// 角色列表：搜索栏 options 与列表 code→name 映射共用一次请求
const roleOptions = ref<RoleInfo[]>([])
const roleMap = computed<Record<string, string>>(() =>
  Object.fromEntries(roleOptions.value.map((r) => [r.code, r.name]))
)
const loadRoles = async () => {
  try {
    const res = await fetchRoleList({ keyword: '', status: '', page: 1, size: 200 })
    roleOptions.value = res?.records ?? []
  } catch {
    // 加载失败静默：列表角色回退显示 code 本身
  }
}

const drawerVisible = ref(false)
const drawerMode = ref<'add' | 'edit' | 'view'>('add')
const editingUser = ref<UserInfo | null>(null)

const openDrawer = (mode: 'add' | 'edit' | 'view', user?: UserInfo) => {
  drawerMode.value = mode
  editingUser.value = user ?? null
  drawerVisible.value = true
}

const onFormSuccess = () => {
  drawerVisible.value = false
  fetchList()
}

onMounted(() => {
  fetchList()
  loadRoles()
})
</script>
