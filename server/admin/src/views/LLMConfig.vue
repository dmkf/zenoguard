<template>
  <div class="llm-config">
    <el-card>
      <template #header>
        <span>LLM配置</span>
      </template>

      <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="模型名称" prop="model_name">
          <el-input
            v-model="form.model_name"
            placeholder="例如: gpt-4, gpt-3.5-turbo"
          />
          <div class="form-tip">
            OpenAI兼容的模型名称
          </div>
        </el-form-item>

        <el-form-item label="API地址" prop="api_url">
          <el-input
            v-model="form.api_url"
            placeholder="例如: https://api.openai.com/v1/chat/completions"
          />
          <div class="form-tip">
            OpenAI兼容的API地址
          </div>
        </el-form-item>

        <el-form-item label="API密钥" prop="api_key">
          <el-input
            v-model="form.api_key"
            type="password"
            show-password
            placeholder="请输入API密钥"
          />
        </el-form-item>

        <el-form-item label="状态" prop="is_active">
          <el-switch v-model="form.is_active" active-text="启用" inactive-text="禁用" />
        </el-form-item>

        <el-divider content-position="left">提示词配置</el-divider>

        <el-form-item label="系统提示词" prop="system_prompt">
          <el-input
            v-model="form.system_prompt"
            type="textarea"
            :rows="4"
            placeholder="定义LLM的角色和行为方式"
          />
          <div class="form-tip">
            定义LLM在分析时的角色定位和行为准则，可使用占位符：{DATA}表示服务器数据，{RULES}表示预警规则
          </div>
        </el-form-item>

        <el-form-item label="用户提示词" prop="user_prompt">
          <el-input
            v-model="form.user_prompt"
            type="textarea"
            :rows="6"
            placeholder="具体分析任务的描述"
          />
          <div class="form-tip">
            指导LLM如何分析数据的具体指令，应包含分析要求和输出格式说明
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSave" :loading="loading">
            保存
          </el-button>
          <el-button @click="handleTest" :loading="testing">
            测试连接
          </el-button>
          <el-button @click="handleResetPrompts">
            重置为默认提示词
          </el-button>
        </el-form-item>
      </el-form>

      <el-divider />

      <el-alert
        title="使用说明"
        type="info"
        :closable="false"
      >
        <ul style="margin: 10px 0 0 20px; padding: 0">
          <li>支持OpenAI兼容的API服务</li>
          <li>LLM将根据预警规则分析服务器数据</li>
          <li>分析结果会记录在上报数据中</li>
          <li>如果分析结果is_alert为true，将触发预警推送</li>
          <li>可自定义提示词来调整分析行为和输出格式</li>
        </ul>
      </el-alert>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { llmApi, type LLMConfig } from '@/api/llm'

const formRef = ref<FormInstance>()
const loading = ref(false)
const testing = ref(false)

// 默认提示词
const defaultSystemPrompt = '你是一个服务器安全分析专家。请根据服务器数据判断是否存在安全问题，并给出简明扼要的分析结果。只返回JSON格式：{"summary": "一句话总结", "is_alert": true/false}'

const defaultUserPrompt = `当前服务器数据：
{DATA}

预警规则：{RULES}

请分析：
1. 用一句话总结服务器状态（不超过50字）
2. 判断是否触发预警（true/false）

只返回JSON格式：{"summary": "一句话总结", "is_alert": true/false}`

const form = reactive({
  model_name: '',
  api_url: '',
  api_key: '',
  is_active: true,
  system_prompt: defaultSystemPrompt,
  user_prompt: defaultUserPrompt
})

const rules = {
  model_name: [{ required: true, message: '请输入模型名称', trigger: 'blur' }],
  api_url: [{ required: true, message: '请输入API地址', trigger: 'blur' }],
  api_key: [{ required: true, message: '请输入API密钥', trigger: 'blur' }]
}

async function loadConfig() {
  loading.value = true
  try {
    const response = await llmApi.get()
    if (response.data) {
      const config = response.data
      form.model_name = config.model_name || ''
      form.api_url = config.api_url || ''
      form.is_active = config.is_active ?? true
      form.system_prompt = config.system_prompt || defaultSystemPrompt
      form.user_prompt = config.user_prompt || defaultUserPrompt
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
      await llmApi.update(form)
      ElMessage.success('保存成功')
    } finally {
      loading.value = false
    }
  })
}

async function handleTest() {
  testing.value = true
  try {
    const response = await llmApi.test()
    if (response.success) {
      ElMessage.success('连接测试成功')
    } else {
      ElMessage.error('连接测试失败: ' + response.message)
    }
  } finally {
    testing.value = false
  }
}

async function handleResetPrompts() {
  try {
    await ElMessageBox.confirm(
      '确定要重置为默认提示词吗？当前的自定义提示词将丢失。',
      '确认重置',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    form.system_prompt = defaultSystemPrompt
    form.user_prompt = defaultUserPrompt
    ElMessage.success('已重置为默认提示词，请点击保存按钮保存更改')
  } catch {
    // 用户取消
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
