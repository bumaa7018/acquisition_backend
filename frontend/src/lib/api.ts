import axios from 'axios'
import { authStorage } from './auth'

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

const api = axios.create({ baseURL: BASE_URL })

api.interceptors.request.use(config => {
  const token = authStorage.getAccessToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  res => res,
  async error => {
    if (error.response?.status === 401) {
      authStorage.clear()
      if (typeof window !== 'undefined') window.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

// ── Types ──────────────────────────────────────────────────────────────────

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}

export interface User {
  id: string
  email: string
  first_name: string
  last_name: string
  roles?: Role[]
  created_at: string
}

export interface Role {
  id: string
  name: string
  permissions: Permission[]
}

export interface Permission {
  id: string
  name: string
}

export interface AdminUnit {
  au3_code: string
  au3_name: string
  au2_code?: string
  au1_code?: string
}

export interface Acquisition {
  id: string
  plan_code: string
  status: number
  area_m2: number
  plan_area_m2?: number
  start_date: string
  end_date: string
  created_at: string
  aus?: AdminUnit[]
  parcels?: Parcel[]
}

export interface Parcel {
  id: string
  parcel_id: string
  au3_code: string
  right_type?: string
  area_m2: number
  acquisition_area_m2: number
}

export interface ParcelDetail {
  detail?: {
    holder_last_name: string
    holder_name: string
    holder_register_no: string
    holder_phone: string
    decision_no: string
    decision_date: string
    certificate_no: string
    certificate_date: string
  }
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  user: User
}

// ── Auth API ───────────────────────────────────────────────────────────────

export const authApi = {
  login: async (email: string, password: string): Promise<LoginResponse> => {
    const { data } = await api.post<LoginResponse>('/api/v1/auth/login', { email, password })
    return data
  },
}

// ── Users API ──────────────────────────────────────────────────────────────

export const usersApi = {
  list: async (params: { page: number; page_size: number }): Promise<PaginatedResponse<User>> => {
    const { data } = await api.get<PaginatedResponse<User>>('/api/v1/users', { params })
    return data
  },
  create: async (body: { email: string; password: string; first_name: string; last_name: string }): Promise<User> => {
    const { data } = await api.post<User>('/api/v1/users', body)
    return data
  },
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/users/${id}`)
  },
}

// ── Roles API ──────────────────────────────────────────────────────────────

export const rolesApi = {
  list: async (): Promise<PaginatedResponse<Role>> => {
    const { data } = await api.get<PaginatedResponse<Role>>('/api/v1/roles')
    return data
  },
  create: async (body: { name: string }): Promise<Role> => {
    const { data } = await api.post<Role>('/api/v1/roles', body)
    return data
  },
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/roles/${id}`)
  },
  assignPermission: async (roleId: string, permId: string): Promise<void> => {
    await api.post(`/api/v1/roles/${roleId}/permissions/${permId}`)
  },
  removePermission: async (roleId: string, permId: string): Promise<void> => {
    await api.delete(`/api/v1/roles/${roleId}/permissions/${permId}`)
  },
}

// ── Permissions API ────────────────────────────────────────────────────────

export const permissionsApi = {
  list: async (): Promise<PaginatedResponse<Permission>> => {
    const { data } = await api.get<PaginatedResponse<Permission>>('/api/v1/permissions')
    return data
  },
}

// ── Land / Acquisition API ─────────────────────────────────────────────────

export const landApi = {
  list: async (params: { page: number; page_size: number; plan_code?: string }): Promise<PaginatedResponse<Acquisition>> => {
    const { data } = await api.get<PaginatedResponse<Acquisition>>('/api/v1/acquisitions', { params })
    return data
  },
  getById: async (id: string): Promise<Acquisition> => {
    const { data } = await api.get<Acquisition>(`/api/v1/acquisitions/${id}`)
    return data
  },
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/acquisitions/${id}`)
  },
  getParcels: async (id: string, params: { page: number; page_size: number }): Promise<PaginatedResponse<Parcel>> => {
    const { data } = await api.get<PaginatedResponse<Parcel>>(`/api/v1/acquisitions/${id}/parcels`, { params })
    return data
  },
  getParcel: async (acquisitionId: string, parcelId: string): Promise<ParcelDetail> => {
    const { data } = await api.get<ParcelDetail>(`/api/v1/acquisitions/${acquisitionId}/parcels/${parcelId}`)
    return data
  },
  syncParcel: async (acquisitionId: string, parcelId: string): Promise<void> => {
    await api.post(`/api/v1/acquisitions/${acquisitionId}/parcels/${parcelId}/sync`)
  },
}
