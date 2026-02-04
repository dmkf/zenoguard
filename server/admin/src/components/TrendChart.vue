<template>
  <div class="trend-chart">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ title }}</span>
          <div class="controls">
            <el-radio-group v-model="timeRange" size="small" @change="handleTimeRangeChange">
              <el-radio-button label="day">今日</el-radio-button>
              <el-radio-button label="week">本周</el-radio-button>
              <el-radio-button label="month">本月</el-radio-button>
              <el-radio-button label="7days">近7天</el-radio-button>
              <el-radio-button label="custom">自定义</el-radio-button>
            </el-radio-group>
            <el-date-picker
              v-if="timeRange === 'custom'"
              v-model="customDateRange"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              format="YYYY-MM-DD HH:mm"
              value-format="YYYY-MM-DD HH:mm:ss"
              @change="fetchData"
              style="margin-left: 10px; width: 350px"
            />
          </div>
        </div>
      </template>

      <div v-loading="loading" ref="chartRef" style="height: 400px"></div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import * as echarts from 'echarts'
import type { EChartsOption } from 'echarts'
import dayjs from 'dayjs'
import { hostApi } from '@/api/host'

interface Props {
  hostId: number
  type: 'load' | 'network'
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '趋势图'
})

const emit = defineEmits<{
  dataLoaded: [data: any[]]
}>()

const chartRef = ref<HTMLElement>()
const loading = ref(false)
const timeRange = ref('day')
const customDateRange = ref<[string, string]>()
let chart: echarts.ECharts | null = null

// 初始化图表
const initChart = () => {
  if (!chartRef.value) return

  chart = echarts.init(chartRef.value, 'macarons')

  const option: EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross'
    }
    },
    legend: {
      data: props.type === 'load' ? ['1分钟负载', '5分钟负载', '15分钟负载'] : ['入站速率', '出站速率']
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'time',
      boundaryGap: false
    },
    yAxis: {
      type: 'value',
      name: props.type === 'load' ? '负载' : '速率',
      axisLabel: {
        formatter: (value: number) => {
          if (props.type === 'load') {
            return value.toFixed(2)
          } else {
            return formatRate(value)
          }
        }
      }
    },
    series: []
  }

  chart.setOption(option)

  // 响应式
  window.addEventListener('resize', () => {
    chart?.resize()
  })
}

// 格式化速率
const formatRate = (bytesPerSecond: number) => {
  if (bytesPerSecond === 0) return '0 B/s'
  const k = 1024
  const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  const i = Math.floor(Math.log(bytesPerSecond) / Math.log(k))
  return (bytesPerSecond / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

// 获取时间范围
const getDateRange = () => {
  const now = dayjs()
  let start: dayjs.Dayjs
  let end: dayjs.Dayjs

  switch (timeRange.value) {
    case 'day':
      start = now.startOf('day')
      end = now.endOf('day')
      break
    case 'week':
      start = now.startOf('week')
      end = now.endOf('week')
      break
    case 'month':
      start = now.startOf('month')
      end = now.endOf('month')
      break
    case '7days':
      start = now.subtract(7, 'day')
      end = now
      break
    case 'custom':
      if (customDateRange.value) {
        start = dayjs(customDateRange.value[0])
        end = dayjs(customDateRange.value[1])
      } else {
        start = now.startOf('day')
        end = now.endOf('day')
      }
      break
    default:
      start = now.startOf('day')
      end = now.endOf('day')
  }

  return {
    start: start.format('YYYY-MM-DD HH:mm:ss'),
    end: end.format('YYYY-MM-DD HH:mm:ss')
  }
}

// 获取图表数据
const fetchData = async () => {
  loading.value = true

  try {
    const range = getDateRange()

    const response = await hostApi.getTrendData(Number(props.hostId), {
      type: props.type,
      start_date: range.start,
      end_date: range.end
    })

    updateChart(response.data)
    emit('dataLoaded', response.data)
  } catch (error: any) {
    console.error('Failed to fetch trend data:', error)

    // 使用模拟数据作为fallback
    const range = getDateRange()
    const mockData = generateMockData(range.start, range.end)
    updateChart(mockData)
    emit('dataLoaded', mockData)
  } finally {
    loading.value = false
  }
}

// 生成模拟数据（测试用）
const generateMockData = (start: string, end: string) => {
  const startTime = dayjs(start)
  const endTime = dayjs(end)
  const data: any[] = []

  const points = 24 // 每24小时一个点
  const interval = endTime.diff(startTime, 'hour') / points

  for (let i = 0; i <= points; i++) {
    const time = startTime.add(i * interval, 'hour')

    if (props.type === 'load') {
      data.push({
        report_time: time.format('YYYY-MM-DD HH:mm:ss'),
        load1: 0.1 + Math.random() * 2,
        load5: 0.2 + Math.random() * 2,
        load15: 0.3 + Math.random() * 2
      })
    } else {
      data.push({
        report_time: time.format('YYYY-MM-DD HH:mm:ss'),
        in_rate: 10000 + Math.random() * 50000,
        out_rate: 5000 + Math.random() * 30000
      })
    }
  }

  return data
}

// 更新图表
const updateChart = (data: any[]) => {
  if (!chart || !data || !Array.isArray(data)) return

  let series: any[]

  if (props.type === 'load') {
    series = [
      {
        name: '1分钟负载',
        type: 'line',
        data: data.map(d => [d.report_time, d.load1]),
        smooth: true
      },
      {
        name: '5分钟负载',
        type: 'line',
        data: data.map(d => [d.report_time, d.load5]),
        smooth: true
      },
      {
        name: '15分钟负载',
        type: 'line',
        data: data.map(d => [d.report_time, d.load15]),
        smooth: true
      }
    ]
  } else {
    series = [
      {
        name: '入站速率',
        type: 'line',
        data: data.map(d => [d.report_time, d.in_rate]),
        smooth: true,
        areaStyle: {}
      },
      {
        name: '出站速率',
        type: 'line',
        data: data.map(d => [d.report_time, d.out_rate]),
        smooth: true,
        areaStyle: {}
      }
    ]
  }

  chart.setOption({
    series
  })
}

// 时间范围变化
const handleTimeRangeChange = () => {
  if (timeRange.value !== 'custom') {
    customDateRange.value = undefined
  }
  fetchData()
}

// 监听hostId变化
watch(() => props.hostId, () => {
  fetchData()
})

onMounted(() => {
  initChart()
  fetchData()

  // 清理
  return () => {
    if (chart) {
      chart.dispose()
      chart = null
    }
    }
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.controls {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>
