<template>
  <div class="host-form">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ isEdit ? '编辑主机' : '新增主机' }}</span>
        </div>
      </template>

      <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="主机名" prop="hostname">
          <el-input v-model="form.hostname" placeholder="例如: server-01" />
        </el-form-item>

        <el-form-item label="备注" prop="remark">
          <el-input
            v-model="form.remark"
            type="textarea"
            :rows="3"
            placeholder="主机备注信息"
          />
        </el-form-item>

        <el-form-item label="上报间隔" prop="report_interval">
          <el-input-number
            v-model="form.report_interval"
            :min="10"
            :max="3600"
            :step="10"
          />
          <span style="margin-left: 10px; color: #909399">秒 (10-3600)</span>
        </el-form-item>

        <el-form-item label="LLM分析间隔" prop="llm_analysis_interval">
          <el-input-number
            v-model="form.llm_analysis_interval"
            :min="300"
            :max="86400"
            :step="300"
          />
          <span style="margin-left: 10px; color: #909399">秒 (300-86400, 默认3600=1小时)</span>
          <div class="form-tip">
            LLM分析时间间隔，不会影响数据上报频率
          </div>
        </el-form-item>

        <el-form-item label="预警规则" prop="alert_rules">
          <el-input
            v-model="form.alert_rules"
            type="textarea"
            :rows="5"
            placeholder="使用自然语言描述预警规则，例如：&#10;1. SSH登录失败超过5次&#10;2. 系统负载超过5.0&#10;3. 网络流量异常增长"
          />
          <div class="form-tip">
            提示：LLM将根据这些规则分析服务器数据并判断是否触发预警
          </div>
        </el-form-item>

        <el-form-item label="状态" prop="is_active">
          <el-switch v-model="form.is_active" active-text="活跃" inactive-text="停用" />
        </el-form-item>

        <el-form-item v-if="isEdit" label="Token">
          <el-input v-model="currentToken" readonly>
            <template #append>
              <el-button @click="copyToken">复制</el-button>
              <el-button @click="regenerateToken">重新生成</el-button>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSubmit" :loading="loading">
            {{ isEdit ? '保存' : '创建' }}
          </el-button>
          <el-button @click="handleCancel">取消</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { hostApi, type Host, type CreateHostRequest, type UpdateHostRequest } from '@/api/host'

const router = useRouter()
const route = useRoute()

const formRef = ref<FormInstance>()
const loading = ref(false)
const currentToken = ref('')
const hostId = ref<number>()

const isEdit = computed(() => !!route.params.id)

const form = reactive({
  hostname: '',
  remark: '',
  report_interval: 300,
  llm_analysis_interval: 3600,
  alert_rules: '',
  is_active: true
})

const rules = {
  hostname: [
    { required: true, message: '请输入主机名', trigger: 'blur' }
  ],
  report_interval: [
    { required: true, message: '请输入上报间隔', trigger: 'blur' }
  ]
}

async function loadHost(id: number) {
  loading.value = true
  try {
    const response = await hostApi.get(id)
    const host = response

    form.hostname = host.hostname
    form.remark = host.remark || ''
    form.report_interval = host.report_interval
    form.llm_analysis_interval = host.llm_analysis_interval || 3600
    form.alert_rules = host.alert_rules || ''
    form.is_active = host.is_active
    currentToken.value = host.token
  } finally {
    loading.value = false
  }
}

async function handleSubmit() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      if (isEdit.value) {
        await hostApi.update(hostId.value!, form as UpdateHostRequest)
        ElMessage.success('保存成功')
      } else {
        await hostApi.create(form as CreateHostRequest)
        ElMessage.success('创建成功')
      }
      router.push('/hosts')
    } finally {
      loading.value = false
    }
  })
}

function handleCancel() {
  router.push('/hosts')
}

function copyToken() {
  navigator.clipboard.writeText(currentToken.value)
  ElMessage.success('Token已复制')
}

async function regenerateToken() {
  try {
    await ElMessageBox.confirm('重新生成Token后，旧Token将失效，是否继续？', '警告', {
      type: 'warning'
    })

    const response = await hostApi.regenerateToken(hostId.value!)
    currentToken.value = response.token
    ElMessage.success('Token已重新生成')
  } catch {
    // Cancelled
  }
}

onMounted(() => {
  if (isEdit.value) {
    hostId.value = Number(route.params.id)
    loadHost(hostId.value)
  }
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-tip {
  margin-top: 5px;
  font-size: 12px;
  color: #909399;
}
</style>
