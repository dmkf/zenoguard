import api from './index'

export interface LoginRequest {
  username: string
  password: string
}

export interface User {
  id: number
  username: string
}

export interface LoginResponse {
  user: User
  token: string
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
  new_password_confirmation: string
}

export const authApi = {
  login(data: LoginRequest) {
    return api.post<any, LoginResponse>('/auth/login', data)
  },

  logout() {
    return api.post('/auth/logout')
  },

  me() {
    return api.get<any, { user: User }>('/auth/me')
  },

  changePassword(data: ChangePasswordRequest) {
    return api.post('/auth/change-password', data)
  }
}
