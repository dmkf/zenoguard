<template>
  <div class="change-password">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>修改密码</span>
        </div>
      </template>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
        style="max-width: 500px"
      >
        <el-form-item label="当前密码" prop="current_password">
          <el-input
            v-model="form.current_password"
            type="password"
            placeholder="请输入当前密码"
            show-password
          />
        </el-form-item>

        <el-form-item label="新密码" prop="new_password">
          <el-input
            v-model="form.new_password"
            type="password"
            placeholder="请输入新密码（至少6位）"
            show-password
          />
        </el-form-item>

        <el-form-item label="确认密码" prop="new_password_confirmation">
          <el-input
            v-model="form.new_password_confirmation"
            type="password"
            placeholder="请再次输入新密码"
            show-password
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSubmit" :loading="loading">
            修改密码
          </el-button>
          <el-button @click="handleCancel">取消</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { authApi, type ChangePasswordRequest } from '@/api/auth'

const router = useRouter()
const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive<ChangePasswordRequest>({
  current_password: '',
  new_password: '',
  new_password_confirmation: ''
})

const validatePass = (rule: any, value: any, callback: any) => {
  if (value === '') {
    callback(new Error('请输入密码'))
  } else if (value.length < 6) {
    callback(new Error('密码长度不能少于6位'))
  } else {
    if (form.new_password_confirmation !== '') {
      formRef.value?.validateField('new_password_confirmation')
    }
    callback()
  }
}

const validatePass2 = (rule: any, value: any, callback: any) => {
  if (value === '') {
    callback(new Error('请再次输入密码'))
  } else if (value !== form.new_password) {
    callback(new Error('两次输入密码不一致'))
  } else {
    callback()
  }
}

const rules: FormRules = {
  current_password: [
    { required: true, message: '请输入当前密码', trigger: 'blur' }
  ],
  new_password: [
    { validator: validatePass, trigger: 'blur' }
  ],
  new_password_confirmation: [
    { validator: validatePass2, trigger: 'blur' }
  ]
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      try {
        await authApi.changePassword(form)
        ElMessage.success('密码修改成功，请重新登录')

        // Logout and redirect to login
        setTimeout(async () => {
          try {
            await authApi.logout()
          } catch (error) {
            // Ignore logout error
          }
          localStorage.removeItem('token')
          router.push('/login')
        }, 1500)
      } catch (error: any) {
        const errorMsg = error.response?.data?.error || '密码修改失败'
        ElMessage.error(errorMsg)
      } finally {
        loading.value = false
      }
    }
  })
}

const handleCancel = () => {
  router.back()
}
</script>

<style scoped>
.change-password {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
