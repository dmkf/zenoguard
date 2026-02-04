<template>
  <div class="llm-summary-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>LLM分析历史 - {{ hostInfo?.hostname }}</span>
          <el-button @click="handleBack">返回</el-button>
        </div>
      </template>

      <el-table :data="summaries" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="分析时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.analysis_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="summary" label="分析结果" min-width="300">
          <template #default="{ row }">
            <div class="summary-cell">{{ row.summary }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.is_alert ? 'danger' : 'success'">
              {{ row.is_alert ? '预警' : '正常' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="fetchSummaries"
        @size-change="fetchSummaries"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { hostApi, type Host, type LLMSummary } from '@/api/host'
import dayjs from 'dayjs'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const hostInfo = ref<Host>()
const summaries = ref<LLMSummary[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

async function fetchHost() {
  try {
    const response = await hostApi.get(Number(route.params.id))
    hostInfo.value = response
  } catch {
    ElMessage.error('获取主机信息失败')
  }
}

async function fetchSummaries() {
  loading.value = true
  try {
    const response = await hostApi.llmSummaries(Number(route.params.id), {
      page: currentPage.value,
      per_page: pageSize.value
    })
    summaries.value = response.data
    total.value = response.total
  } finally {
    loading.value = false
  }
}

function handleBack() {
  router.push(`/hosts/${route.params.id}/data`)
}

function formatDateTime(date: string) {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

onMounted(() => {
  fetchHost()
  fetchSummaries()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.summary-cell {
  max-width: 500px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  line-height: 1.6;
}
</style>
