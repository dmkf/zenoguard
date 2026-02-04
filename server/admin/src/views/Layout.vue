<template>
  <el-container class="layout-container">
    <el-aside width="200px">
      <div class="logo">
        <h3>智巡Guard</h3>
      </div>
      <el-menu
        :default-active="activeMenu"
        router
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
      >
        <el-menu-item index="/dashboard">
          <el-icon><Platform /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>
        <el-menu-item index="/hosts">
          <el-icon><Monitor /></el-icon>
          <span>主机管理</span>
        </el-menu-item>
        <el-menu-item index="/llm">
          <el-icon><ChatDotRound /></el-icon>
          <span>LLM配置</span>
        </el-menu-item>
        <el-menu-item index="/alert">
          <el-icon><Bell /></el-icon>
          <span>预警配置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header>
        <div class="header-content">
          <span>{{ pageTitle }}</span>
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-icon><User /></el-icon>
              {{ authStore.user?.username }}
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="changePassword">修改密码</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const activeMenu = computed(() => route.path)
const pageTitle = computed(() => {
  const titles: Record<string, string> = {
    '/dashboard': '仪表盘',
    '/hosts': '主机管理',
    '/llm': 'LLM配置',
    '/alert': '预警配置',
    '/change-password': '修改密码'
  }
  return titles[route.path] || '智巡Guard'
})

async function handleCommand(command: string) {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
        type: 'warning'
      })
      await authStore.logout()
      ElMessage.success('已退出登录')
      router.push('/login')
    } catch {
      // Cancelled
    }
  } else if (command === 'changePassword') {
    router.push('/change-password')
  }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.el-aside {
  background-color: #304156;
}

.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  background-color: #2b3a4a;
}

.logo h3 {
  margin: 0;
  color: #fff;
  font-size: 18px;
}

.el-header {
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  padding: 0 20px;
}

.header-content {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.user-info {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 5px;
}

.el-main {
  background-color: #f5f5f5;
  padding: 20px;
}
</style>
