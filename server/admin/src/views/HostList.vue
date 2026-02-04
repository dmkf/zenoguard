<template>
  <div class="host-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>主机列表</span>
          <el-button type="primary" @click="handleCreate">
            <el-icon><Plus /></el-icon>
            新增主机
          </el-button>
        </div>
      </template>

      <el-table :data="hosts" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="hostname" label="主机名" />
        <el-table-column label="Token" width="300">
          <template #default="{ row }">
            <el-input
              v-model="row.token"
              type="text"
              readonly
              size="small"
            >
              <template #append>
                <el-button @click="copyToken(row.token)">复制</el-button>
              </template>
            </el-input>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" />
        <el-table-column prop="report_interval" label="上报间隔(秒)" width="120" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'info'">
              {{ row.is_active ? '活跃' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="最后上报" width="180">
          <template #default="{ row }">
            {{ row.latest_data ? formatDate(row.latest_data.report_time) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleViewData(row)">数据</el-button>
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hostApi, type Host } from '@/api/host'
import dayjs from 'dayjs'

const router = useRouter()
const loading = ref(false)
const hosts = ref<Host[]>([])

async function fetchHosts() {
  loading.value = true
  try {
    const response = await hostApi.list()
    hosts.value = response.data
  } finally {
    loading.value = false
  }
}

function handleCreate() {
  router.push('/hosts/create')
}

function handleEdit(host: Host) {
  router.push(`/hosts/${host.id}/edit`)
}

function handleViewData(host: Host) {
  router.push(`/hosts/${host.id}/data`)
}

async function handleDelete(host: Host) {
  try {
    await ElMessageBox.confirm(`确定要删除主机 "${host.hostname}" 吗？此操作不可恢复！`, '警告', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    // Ask for password confirmation
    try {
      const { value: password } = await ElMessageBox.prompt(
        '为了防止误操作，请输入管理员密码确认删除',
        '安全验证',
        {
          confirmButtonText: '确认删除',
          cancelButtonText: '取消',
          inputType: 'password',
          inputPattern: /.+/,
          inputErrorMessage: '请输入密码',
          inputPlaceholder: '请输入管理员密码'
        }
      )

      // Delete with password confirmation
      await hostApi.delete(host.id, { password })
      ElMessage.success('删除成功')
      fetchHosts()
    } catch {
      // Password confirmation cancelled
      ElMessage.info('已取消删除')
    }
  } catch {
    // First confirmation cancelled
  }
}

function copyToken(token: string) {
  navigator.clipboard.writeText(token)
  ElMessage.success('Token已复制')
}

function formatDate(date: string) {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

onMounted(() => {
  fetchHosts()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
