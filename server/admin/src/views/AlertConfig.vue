<template>
  <div class="alert-config">
    <el-card>
      <template #header>
        <span>钉钉预警配置</span>
      </template>

      <el-form :model="form" :rules="rules" ref="formRef" label-width="140px">
        <el-form-item label="Webhook地址" prop="webhook_url">
          <el-input
            v-model="form.webhook_url"
            placeholder="钉钉机器人Webhook地址"
          />
          <div class="form-tip">
            在钉钉群设置中添加自定义机器人获取Webhook地址
          </div>
        </el-form-item>

        <el-form-item label="签名密钥" prop="secret">
          <el-input
            v-model="form.secret"
            type="password"
            show-password
            placeholder="可选：钉钉机器人签名密钥"
          />
          <div class="form-tip">
            如果启用了签名验证，请填写密钥
          </div>
        </el-form-item>

        <el-form-item label="状态" prop="is_active">
          <el-switch v-model="form.is_active" active-text="启用" inactive-text="禁用" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSave" :loading="loading">
            保存
          </el-button>
          <el-button @click="handleTest" :loading="testing">
            发送测试
          </el-button>
        </el-form-item>
      </el-form>

      <el-divider />

      <el-alert
        title="配置说明"
        type="info"
        :closable="false"
      >
        <ul style="margin: 10px 0 0 20px; padding: 0">
          <li>在钉钉群设置中添加"自定义"机器人</li>
          <li>安全设置可选择"自定义关键词"或"加签"</li>
          <li>如果使用"加签"，必须填写签名密钥</li>
          <li>关键词建议使用：智巡 或 预警</li>
        </ul>
      </el-alert>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { alertApi, type AlertConfig } from '@/api/alert'

const formRef = ref<FormInstance>()
const loading = ref(false)
const testing = ref(false)

const form = reactive({
  platform: 'dingtalk',
  webhook_url: '',
  secret: '',
  is_active: true
})

const rules = {
  webhook_url: [{ required: true, message: '请输入Webhook地址', trigger: 'blur' }]
}

async function loadConfig() {
  loading.value = true
  try {
    const response = await alertApi.list()
    const configs = response.data
    const dingtalkConfig = configs.find((c: AlertConfig) => c.platform === 'dingtalk')

    if (dingtalkConfig) {
      form.webhook_url = dingtalkConfig.webhook_url
      form.is_active = dingtalkConfig.is_active
    }
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      await alertApi.update(form)
      ElMessage.success('保存成功')
    } finally {
      loading.value = false
    }
  })
}

async function handleTest() {
  testing.value = true
  try {
    const response = await alertApi.test('dingtalk')
    if (response.success) {
      ElMessage.success('测试消息已发送')
    } else {
      ElMessage.error('发送失败: ' + response.message)
    }
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.form-tip {
  margin-top: 5px;
  font-size: 12px;
  color: #909399;
}
</style>
