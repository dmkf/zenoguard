<template>
  <div class="host-data-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>主机数据 - {{ hostInfo?.hostname }}</span>
          <el-button @click="handleBack">返回</el-button>
        </div>
      </template>

      <!-- 趋势图表 -->
      <el-row :gutter="20" style="margin-bottom: 20px">
        <el-col :span="12">
          <TrendChart :host-id="Number(route.params.id)" type="load" title="系统负载趋势" />
        </el-col>
        <el-col :span="12">
          <TrendChart :host-id="Number(route.params.id)" type="network" title="网络流量速率趋势" />
        </el-col>
      </el-row>

      <!-- LLM分析结果 -->
      <el-card v-if="latestLLMSummary" style="margin-bottom: 20px" :class="{ 'alert-card': latestLLMSummary.is_alert }">
        <template #header>
          <div class="card-header">
            <span>
              <el-icon><ChatLineSquare /></el-icon>
              最新LLM分析
            </span>
            <div>
              <el-tag :type="latestLLMSummary.is_alert ? 'danger' : 'success'" style="margin-right: 10px">
                {{ latestLLMSummary.is_alert ? '触发预警' : '正常' }}
              </el-tag>
              <el-button size="small" :loading="analyzing" @click="handleTriggerAnalysis">
                <el-icon><Refresh /></el-icon>
                立即分析
              </el-button>
              <el-button size="small" @click="handleViewLLMHistory">历史记录</el-button>
            </div>
          </div>
        </template>
        <div class="llm-summary-content">
          <p class="analysis-time">{{ formatDateTime(latestLLMSummary.analysis_time) }}</p>
          <p class="summary-text">{{ latestLLMSummary.summary }}</p>
        </div>
      </el-card>
      <el-card v-else style="margin-bottom: 20px">
        <template #header>
          <div class="card-header">
            <span>
              <el-icon><ChatLineSquare /></el-icon>
              LLM分析
            </span>
            <el-button size="small" :loading="analyzing" @click="handleTriggerAnalysis">
              <el-icon><Refresh /></el-icon>
              立即分析
            </el-button>
          </div>
        </template>
        <el-empty description="暂无LLM分析记录，点击'立即分析'开始" :image-size="80" />
      </el-card>

      <!-- 筛选表单 -->
      <el-form :inline="true" class="filter-form">
        <el-form-item label="日期范围">
          <el-date-picker
            v-model="dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            @change="fetchData"
          />
        </el-form-item>

        <el-form-item label="预警状态">
          <el-select v-model="isAlert" clearable placeholder="全部" @change="fetchData">
            <el-option label="正常" :value="false" />
            <el-option label="预警" :value="true" />
          </el-select>
        </el-form-item>

        <el-form-item label="搜索">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索关键词"
            clearable
            @keyup.enter="fetchData"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="fetchData">查询</el-button>
        </el-form-item>

        <el-form-item style="margin-left: auto;">
          <el-dropdown @command="handleCleanData" style="margin-right: 10px;">
            <el-button type="warning" :loading="cleaning">
              <el-icon><Delete /></el-icon>
              数据清理
              <el-icon style="margin-left: 5px;"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="3d">
                  <el-icon><Calendar /></el-icon>
                  清理3天以前的数据
                </el-dropdown-item>
                <el-dropdown-item command="1w">
                  <el-icon><Calendar /></el-icon>
                  清理1周以前的数据
                </el-dropdown-item>
                <el-dropdown-item command="1m">
                  <el-icon><Calendar /></el-icon>
                  清理1个月以前的数据
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </el-form-item>
      </el-form>

      <el-table :data="dataList" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="report_time" label="上报时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.report_time) }}
          </template>
        </el-table-column>
        <el-table-column label="公网IP" width="250">
          <template #default="{ row }">
            <div>{{ row.public_ip || '-' }}</div>
            <div style="font-size: 12px; color: #999;">{{ row.ip_location || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="系统负载" width="150">
          <template #default="{ row }">
            <span v-if="row.system_load">
              {{ row.system_load.load1 }}, {{ row.system_load.load5 }}, {{ row.system_load.load15 }}
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="网络流量速率" width="150">
          <template #default="{ row }">
            <div v-if="row.network_traffic">
              <div>↓ {{ formatRate(row.network_traffic.total_in_bytes) }}</div>
              <div>↑ {{ formatRate(row.network_traffic.total_out_bytes) }}</div>
            </div>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="SSH登录" width="250">
          <template #default="{ row }">
            <div v-if="row.ssh_logins && row.ssh_logins.length > 0">
              <div v-for="(login, index) in row.ssh_logins.slice(0, 3)" :key="index" style="margin-bottom: 6px;">
                <div style="font-size: 13px;">
                  {{ login.user }}@{{ login.ip }}
                  <el-tag v-if="login.is_active" type="success" size="small" style="margin-left: 5px">活跃</el-tag>
                  <span v-if="login.session_duration > 0" style="color: #606266; font-size: 12px; margin-left: 5px">
                    {{ formatDuration(login.session_duration) }}
                  </span>
                </div>
                <div style="font-size: 11px; color: #909399; margin-top: 1px;">
                  <span v-if="login.ip_location">{{ login.ip_location }}</span>
                  <span v-if="login.ip_location && login.time"> · </span>
                  <span v-if="login.time">{{ formatDateTime(login.time) }}</span>
                </div>
              </div>
              <div v-if="row.ssh_logins.length > 3" style="color: #999; font-size: 12px; margin-top: 4px;">
                还有 {{ row.ssh_logins.length - 3 }} 条...
              </div>
            </div>
            <span v-else style="color: #999; font-size: 13px;">无记录</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleViewDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="fetchData"
        @size-change="fetchData"
      />
    </el-card>

    <!-- Detail Dialog -->
    <el-dialog v-model="detailVisible" title="数据详情" width="800px">
      <el-descriptions :column="2" border v-if="currentData">
        <el-descriptions-item label="上报时间">
          {{ formatDate(currentData.report_time) }}
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="currentData.is_alert ? 'danger' : 'success'">
            {{ currentData.is_alert ? '预警' : '正常' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="公网IP" :span="2">
          <div>{{ currentData.public_ip || '-' }}</div>
          <div style="font-size: 12px; color: #999;">{{ currentData.ip_location || '-' }}</div>
        </el-descriptions-item>

        <el-descriptions-item label="系统负载" :span="2" v-if="currentData.system_load">
          1分钟: {{ currentData.system_load.load1 }} |
          5分钟: {{ currentData.system_load.load5 }} |
          15分钟: {{ currentData.system_load.load15 }}
        </el-descriptions-item>

        <el-descriptions-item label="网络流量速率" :span="2" v-if="currentData.network_traffic">
          接口: {{ currentData.network_traffic.interface }} |
          入站: {{ formatRate(currentData.network_traffic.total_in_bytes) }} |
          出站: {{ formatRate(currentData.network_traffic.total_out_bytes) }} |
          采样数: {{ currentData.network_traffic.sample_count }}
        </el-descriptions-item>

        <el-descriptions-item label="SSH登录" :span="2">
          <div v-if="currentData.ssh_logins && currentData.ssh_logins.length > 0">
            <div v-for="(login, index) in currentData.ssh_logins" :key="index" style="margin-bottom: 8px; padding: 10px; background: #f5f7fa; border-radius: 4px;">
              <div style="font-size: 14px; font-weight: 500; margin-bottom: 4px;">
                {{ login.user }}@{{ login.ip }}
                <span style="color: #909399; font-weight: normal; font-size: 13px; margin-left: 8px;">
                  ({{ login.success ? '成功' : '失败' }})
                </span>
                <el-tag v-if="login.is_active" type="success" size="small" style="margin-left: 8px">活跃</el-tag>
              </div>
              <div style="font-size: 12px; color: #606266; line-height: 1.8;">
                <div v-if="login.ip_location" style="margin-bottom: 3px;">
                  <span style="color: #909399;">位置:</span> {{ login.ip_location }}
                </div>
                <div style="display: flex; gap: 20px;">
                  <span v-if="login.time">
                    <span style="color: #909399;">时间:</span> {{ formatDateTime(login.time) }}
                  </span>
                  <span v-if="login.session_duration > 0">
                    <span style="color: #909399;">时长:</span> {{ formatDuration(login.session_duration) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
          <div v-else style="color: #999;">
            无SSH登录记录
          </div>
        </el-descriptions-item>
      </el-descriptions>

      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hostApi, type Host, type HostData, type LLMSummary } from '@/api/host'
import dayjs from 'dayjs'
import TrendChart from '@/components/TrendChart.vue'
import { ChatLineSquare, Refresh, Delete, ArrowDown, Calendar } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const analyzing = ref(false)
const cleaning = ref(false)
const hostInfo = ref<Host>()
const latestLLMSummary = ref<LLMSummary>()
const dataList = ref<HostData[]>([])
const currentPage = ref(1)
const pageSize = ref(50)
const total = ref(0)
const dateRange = ref<[string, string]>()
const isAlert = ref<boolean | undefined>()
const searchKeyword = ref('')
const detailVisible = ref(false)
const currentData = ref<HostData>()

async function fetchHost() {
  try {
    const response = await hostApi.get(Number(route.params.id))
    hostInfo.value = response
    // Load latest LLM summary (handle snake_case from backend)
    if (response.latest_l_l_m_summary) {
      latestLLMSummary.value = response.latest_l_l_m_summary
    }
  } catch {
    ElMessage.error('获取主机信息失败')
  }
}

async function fetchData() {
  loading.value = true
  try {
    const params: any = {
      page: currentPage.value,
      per_page: pageSize.value
    }

    if (dateRange.value) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }

    if (isAlert.value !== undefined) {
      params.is_alert = isAlert.value
    }

    if (searchKeyword.value) {
      params.search = searchKeyword.value
    }

    const response = await hostApi.dataList(Number(route.params.id), params)
    dataList.value = response.data
    total.value = response.total
  } finally {
    loading.value = false
  }
}

function handleBack() {
  router.push('/hosts')
}

function handleViewDetail(row: HostData) {
  currentData.value = row
  detailVisible.value = true
}

function formatDate(date: string) {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

function formatBytes(bytes: number) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

function formatRate(bytesPerSecond: number) {
  if (bytesPerSecond === 0) return '0 B/s'
  const k = 1024
  const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s', 'TB/s']
  const i = Math.floor(Math.log(bytesPerSecond) / Math.log(k))
  return Math.round(bytesPerSecond / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

function formatDuration(seconds: number) {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}小时`
  return `${Math.floor(seconds / 86400)}天`
}

function formatDateTime(date: string) {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

function handleViewLLMHistory() {
  router.push(`/hosts/${route.params.id}/llm-summaries`)
}

async function handleTriggerAnalysis() {
  analyzing.value = true
  try {
    const response = await hostApi.triggerAnalysis(Number(route.params.id))
    ElMessage.success('LLM分析已完成')
    // Refresh host info to get latest summary
    await fetchHost()
  } catch (error: any) {
    const errorMsg = error.response?.data?.error || '分析失败'
    ElMessage.error(errorMsg)
  } finally {
    analyzing.value = false
  }
}

async function handleCleanData(command: string) {
  const timeTextMap: Record<string, string> = {
    '3d': '3天',
    '1w': '1周',
    '1m': '1个月'
  }

  const timeText = timeTextMap[command]
  if (!timeText) return

  try {
    await ElMessageBox.confirm(
      `确定要清理${timeText}以前的所有主机数据吗？此操作不可恢复！`,
      '数据清理确认',
      {
        confirmButtonText: '确定清理',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )

    cleaning.value = true
    try {
      const response = await hostApi.cleanOldData(Number(route.params.id), command)
      ElMessage.success(`清理完成，删除了 ${response.deleted_count} 条数据`)
      // Refresh the data list
      await fetchData()
    } catch (error: any) {
      const errorMsg = error.response?.data?.error || error.response?.data?.message || '清理失败'
      ElMessage.error(errorMsg)
    } finally {
      cleaning.value = false
    }
  } catch {
    // User cancelled
  }
}

onMounted(() => {
  fetchHost()
  fetchData()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-form {
  margin-bottom: 20px;
}

.llm-summary-content {
  line-height: 1.8;
}

.llm-summary-content .analysis-time {
  color: #909399;
  font-size: 13px;
  margin-bottom: 10px;
}

.llm-summary-content .summary-text {
  font-size: 15px;
  color: #303133;
  white-space: pre-wrap;
  word-break: break-word;
}

.alert-card {
  border: 2px solid #f56c6c;
  background-color: #fef0f0;
}
</style>
