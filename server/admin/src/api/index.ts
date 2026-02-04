import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  timeout: 30000
})

// Request interceptor
api.interceptors.request.use(
  config => {
    const authStore = useAuthStore()
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  response => {
    return response.data
  },
  error => {
    const authStore = useAuthStore()

    if (error.response) {
      const { status, data } = error.response

      if (status === 401) {
        authStore.logout()
        window.location.href = '/admin/login'
        ElMessage.error('请先登录')
      } else if (status === 403) {
        ElMessage.error('没有权限')
      } else if (status === 500) {
        ElMessage.error('服务器错误')
      } else {
        ElMessage.error(data.error || data.message || '请求失败')
      }
    } else {
      ElMessage.error('网络错误')
    }

    return Promise.reject(error)
  }
)

export default api
