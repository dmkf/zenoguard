import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/views/Layout.vue'),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue')
      },
      {
        path: 'hosts',
        name: 'HostList',
        component: () => import('@/views/HostList.vue')
      },
      {
        path: 'hosts/create',
        name: 'HostCreate',
        component: () => import('@/views/HostForm.vue')
      },
      {
        path: 'hosts/:id/edit',
        name: 'HostEdit',
        component: () => import('@/views/HostForm.vue')
      },
      {
        path: 'hosts/:id/data',
        name: 'HostData',
        component: () => import('@/views/HostDataList.vue')
      },
      {
        path: 'hosts/:id/llm-summaries',
        name: 'LLMSummaries',
        component: () => import('@/views/LLMSummaryList.vue')
      },
      {
        path: 'llm',
        name: 'LLMConfig',
        component: () => import('@/views/LLMConfig.vue')
      },
      {
        path: 'alert',
        name: 'AlertConfig',
        component: () => import('@/views/AlertConfig.vue')
      },
      {
        path: 'change-password',
        name: 'ChangePassword',
        component: () => import('@/views/ChangePassword.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL || '/admin/'),
  routes
})

// Navigation guard
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  const requiresAuth = to.meta.requiresAuth !== false

  if (requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/')
  } else {
    next()
  }
})

export default router
