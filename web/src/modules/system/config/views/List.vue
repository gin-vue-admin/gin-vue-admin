<template>
  <SearchTable
    title="参数配置"
    :loading="loading"
    :data="tableData"
    :columns="columns"
    :pagination="pagination"
    @search="handleSearch"
    @reset="handleReset"
    @page-change="handlePageChange"
  >
    <template #search>
      <el-input
        v-model="searchForm.keyword"
        placeholder="配置 Key / 名称"
        clearable
        style="width: 220px"
        @keyup.enter="handleSearch"
      />
    </template>

    <template #actions>
      <el-button
        type="primary"
        :icon="Plus"
        @click="openDrawer('add')"
      >
        新增配置
      </el-button>
      <el-button
        :icon="Refresh"
        @click="fetchList"
      >
        刷新
      </el-button>
    </template>

    <template #col-configValue="{ row }">
      <span class="config-value">{{ String(row.configValue ?? '').slice(0, 60) }}{{ String(row.configValue ?? '').length > 60 ? '…' : '' }}</span>
    </template>

    <template #col-updateTime="{ row }">
      {{ formatDate(row.updateTime) }}
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

  <ConfigFormDrawer
    v-model="drawerVisible"
    :mode="drawerMode"
    :data="editingRow"
    @success="onFormSuccess"
  />
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { Plus, Refresh } from '@element-plus/icons-vue'
import { SearchTable } from '@/app/components'
import { useCrud } from '@/app/composables/useCrud'
import { formatDate } from '@/lib/format'
import type { ColumnDef } from '@/app/components/SearchTable/types'
import {
  fetchSysConfigList,
  deleteSysConfig,
  type SysConfigInfo,
  type SysConfigSearchRequest,
} from '../api'
import ConfigFormDrawer from './ConfigFormDrawer.vue'

const {
  listData,
  loading,
  pagination,
  searchForm,
  fetchList,
  handleSearch,
  handleReset,
  handlePageChange,
  handleDelete,
} = useCrud<SysConfigInfo>({
  fetch: (params) => fetchSysConfigList(params as unknown as SysConfigSearchRequest),
  remove: deleteSysConfig,
  defaultSearchForm: { keyword: '' },
  pageSize: 10,
})

const tableData = computed(() => listData.value as unknown as Record<string, unknown>[])

const columns: ColumnDef[] = [
  { prop: 'configKey', label: '配置 Key', minWidth: 180 },
  { prop: 'configName', label: '名称', minWidth: 140 },
  { prop: 'configValue', label: '值', minWidth: 220, slot: 'configValue' },
  { prop: 'type', label: '类型', minWidth: 100 },
  { prop: 'remark', label: '备注', minWidth: 160 },
  { prop: 'updateTime', label: '更新时间', minWidth: 170, slot: 'updateTime' },
  { prop: 'actions', label: '操作', width: 180, fixed: 'right', slot: 'actions' },
]

const drawerVisible = ref(false)
const drawerMode = ref<'add' | 'edit' | 'view'>('add')
const editingRow = ref<SysConfigInfo | null>(null)

const openDrawer = (mode: 'add' | 'edit' | 'view', row?: SysConfigInfo) => {
  drawerMode.value = mode
  editingRow.value = row ?? null
  drawerVisible.value = true
}

const onFormSuccess = () => {
  drawerVisible.value = false
  fetchList()
}

onMounted(fetchList)
</script>

<style scoped>
.config-value {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12px;
  color: var(--el-text-color-regular);
  word-break: break-all;
}
</style>
