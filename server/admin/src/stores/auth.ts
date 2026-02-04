import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi, type User } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<User | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  async function login(username: string, password: string) {
    try {
      const response = await authApi.login({ username, password })
      token.value = response.token
      user.value = response.user
      localStorage.setItem('token', response.token)
      return true
    } catch (error) {
      return false
    }
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      token.value = null
      user.value = null
      localStorage.removeItem('token')
    }
  }

  async function fetchUser() {
    if (!token.value) return false

    try {
      const response = await authApi.me()
      user.value = response.user
      return true
    } catch (error) {
      // Token might be invalid, clear it
      token.value = null
      user.value = null
      localStorage.removeItem('token')
      return false
    }
  }

  // Initialize: fetch user info on store creation if token exists
  if (token.value && !user.value) {
    fetchUser()
  }

  return {
    token,
    user,
    isAuthenticated,
    login,
    logout,
    fetchUser
  }
})
