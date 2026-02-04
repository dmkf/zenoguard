<template>
  <div class="dashboard">
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #409EFF">
              <el-icon size="30"><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total_hosts }}</div>
              <div class="stat-label">主机总数</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #67C23A">
              <el-icon size="30"><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.active_hosts }}</div>
              <div class="stat-label">活跃主机</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #E6A23C">
              <el-icon size="30"><Warning /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.alerts_today }}</div>
              <div class="stat-label">今日预警</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #F56C6C">
              <el-icon size="30"><Document /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total_reports }}</div>
              <div class="stat-label">上报总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>最近LLM分析</span>
            </div>
          </template>
          <el-table :data="recentAnalyses" v-loading="loading">
            <el-table-column prop="hostname" label="主机" width="150" />
            <el-table-column prop="analysis_time" label="分析时间" width="180">
              <template #default="{ row }">
                {{ formatDate(row.analysis_time) }}
              </template>
            </el-table-column>
            <el-table-column prop="summary" label="分析结果" show-overflow-tooltip />
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag v-if="row.is_alert" type="danger">预警</el-tag>
                <el-tag v-else type="success">正常</el-tag>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loading && recentAnalyses.length === 0" description="暂无LLM分析记录" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { hostApi } from '@/api/host'
import dayjs from 'dayjs'
import axios from 'axios'

const loading = ref(false)
const stats = ref({
  total_hosts: 0,
  active_hosts: 0,
  alerts_today: 0,
  total_reports: 0
})
const recentAnalyses = ref<any[]>([])

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  timeout: 30000
})

// Get token from localStorage
const token = localStorage.getItem('token')
if (token) {
  api.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

async function fetchData() {
  loading.value = true
  try {
    // Fetch dashboard stats
    const statsResponse = await api.get('/dashboard/stats')
    stats.value = statsResponse.data

    // Fetch recent LLM analyses
    try {
      const analysesResponse = await api.get('/llm-analyses/recent?limit=10')
      recentAnalyses.value = analysesResponse.data.data || []
    } catch (error) {
      console.error('Failed to fetch LLM analyses:', error)
      recentAnalyses.value = []
    }

  } finally {
    loading.value = false
  }
}

function formatDate(date: string) {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.stat-card {
  cursor: pointer;
  transition: transform 0.2s;
}

.stat-card:hover {
  transform: translateY(-2px);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 15px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
