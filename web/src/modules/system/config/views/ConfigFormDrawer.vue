<template>
  <FormDrawer
    v-model="visible"
    :title="drawerTitle"
    :mode="mode"
    :form-data="formData"
    :fields="fields"
    :rules="rules"
    :loading="submitting"
    width="520px"
    @submit="handleSubmit"
  />
</template>

<script lang="ts" setup>
import { ref, watch, reactive, computed } from 'vue'
import { FormDrawer } from '@/app/components'
import type { FormField, FormDrawerMode } from '@/app/components/FormDrawer/types'
import { ElMessage } from 'element-plus'
import {
  createSysConfig,
  updateSysConfig,
  type SysConfigInfo,
  type SysConfigUpsertRequest,
} from '../api'

const props = defineProps<{
  modelValue: boolean
  mode: FormDrawerMode
  data: SysConfigInfo | null
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  success: []
}>()

const visible = ref(props.modelValue)
const submitting = ref(false)

watch(() => props.modelValue, (v) => {
  visible.value = v
  if (v) initForm()
})
watch(visible, (v) => emit('update:modelValue', v))

const formData = reactive<SysConfigUpsertRequest>({
  configKey: '',
  configValue: '',
  configName: '',
  remark: '',
  type: '',
})

const drawerTitle = computed(() => {
  if (props.mode === 'add') return '新增配置'
  if (props.mode === 'edit') return '编辑配置'
  return '查看配置'
})

const fields: FormField[] = [
  { prop: 'configKey', label: '配置 Key', type: 'input', span: 24, placeholder: '全局唯一，如 site_title' },
  { prop: 'configName', label: '名称', type: 'input', span: 24 },
  { prop: 'configValue', label: '值', type: 'textarea', span: 24 },
  { prop: 'type', label: '类型', type: 'input', span: 12, placeholder: 'string / int / bool' },
  { prop: 'remark', label: '备注', type: 'textarea', span: 24 },
]

const rules = {
  configKey: [
    { required: true, message: '请输入配置 Key', trigger: 'blur' },
    { max: 128, message: 'Key 长度不能超过 128', trigger: 'blur' },
  ],
  configName: [{ max: 128, message: '名称长度不能超过 128', trigger: 'blur' }],
}

const initForm = () => {
  if ((props.mode === 'edit' || props.mode === 'view') && props.data) {
    Object.assign(formData, {
      configKey: props.data.configKey,
      configValue: props.data.configValue,
      configName: props.data.configName,
      remark: props.data.remark,
      type: props.data.type,
    })
  } else {
    Object.assign(formData, {
      configKey: '',
      configValue: '',
      configName: '',
      remark: '',
      type: '',
    })
  }
}

const handleSubmit = async (data: Record<string, unknown>) => {
  submitting.value = true
  try {
    const payload = { ...formData, ...data } as SysConfigUpsertRequest
    if (props.mode === 'add') {
      await createSysConfig(payload)
      ElMessage.success('新增成功')
    } else {
      await updateSysConfig(props.data!.id, payload)
      ElMessage.success('更新成功')
    }
    emit('success')
    emit('update:modelValue', false)
  } finally {
    submitting.value = false
  }
}
</script>
