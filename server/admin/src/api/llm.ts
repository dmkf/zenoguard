import api from './index'

export interface LLMConfig {
  id: number
  model_name: string
  api_url: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface UpdateLLMRequest {
  model_name: string
  api_url: string
  api_key: string
  is_active?: boolean
}

export const llmApi = {
  get() {
    return api.get<any, { data: LLMConfig | null }>('/llm')
  },

  update(data: UpdateLLMRequest) {
    return api.put<any, any>('/llm', data)
  },

  test() {
    return api.post<any, any>('/llm/test')
  }
}
