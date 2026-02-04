import api from './index'

export interface Host {
  id: number
  hostname: string
  token: string
  remark?: string
  report_interval: number
  llm_analysis_interval: number
  alert_rules?: string
  is_active: boolean
  created_at: string
  updated_at: string
  latest_data?: any
  latest_l_l_m_summary?: any  // Changed from latestLLMSummary
}

export interface HostData {
  id: number
  host_id: number
  report_time: string
  ssh_logins?: any[]
  system_load?: any
  network_traffic?: any
  public_ip?: string
  ip_location?: string
  llm_summary?: string
  is_alert: boolean
  created_at: string
}

export interface LLMSummary {
  id: number
  host_id: number
  summary: string
  is_alert: boolean
  analysis_time: string
  created_at: string
  updated_at: string
}

export interface CreateHostRequest {
  hostname: string
  remark?: string
  report_interval?: number
  llm_analysis_interval?: number
  alert_rules?: string
  is_active?: boolean
}

export interface UpdateHostRequest {
  hostname?: string
  remark?: string
  report_interval?: number
  llm_analysis_interval?: number
  alert_rules?: string
  is_active?: boolean
}

export const hostApi = {
  list(params?: any) {
    return api.get<any, any>('/hosts', { params })
  },

  get(id: number) {
    return api.get<any, Host>(`/hosts/${id}`)
  },

  create(data: CreateHostRequest) {
    return api.post<any, Host>('/hosts', data)
  },

  update(id: number, data: UpdateHostRequest) {
    return api.put<any, Host>(`/hosts/${id}`, data)
  },

  delete(id: number, data?: { password?: string }) {
    return api.delete(`/hosts/${id}`, { data })
  },

  dataList(id: number, params?: any) {
    return api.get<any, any>(`/hosts/${id}/data`, { params })
  },

  regenerateToken(id: number) {
    return api.post<any, { token: string }>(`/hosts/${id}/regenerate-token`)
  },

  getTrendData(id: number, params: { type: string; start_date: string; end_date: string }) {
    return api.get<any, any>(`/hosts/${id}/trend`, { params })
  },

  llmSummaries(id: number, params?: any) {
    return api.get<any, any>(`/hosts/${id}/llm-summaries`, { params })
  },

  triggerAnalysis(id: number) {
    return api.post<any, any>(`/hosts/${id}/trigger-analysis`)
  },

  cleanOldData(id: number, period: string) {
    return api.delete<any, { deleted_count: number }>(`/hosts/${id}/data/clean`, {
      params: { period }
    })
  }
}
