import api from './index'

export interface AlertConfig {
  id: number
  platform: string
  webhook_url: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface UpdateAlertRequest {
  platform: string
  webhook_url: string
  secret?: string
  is_active?: boolean
}

export const alertApi = {
  list() {
    return api.get<any, { data: AlertConfig[] }>('/alert')
  },

  update(data: UpdateAlertRequest) {
    return api.put<any, any>('/alert', data)
  },

  test(platform: string = 'dingtalk') {
    return api.post<any, any>('/alert/test', { platform })
  },

  delete(id: number) {
    return api.delete(`/alert/${id}`)
  }
}
