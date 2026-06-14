import axios from 'axios'
import { useAuthStore } from '@/store/auth'
import router from '@/router'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: attach access token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

let isRefreshing = false
let failedQueue = []

const processQueue = (error, token = null) => {
  failedQueue.forEach((prom) => {
    if (error) prom.reject(error)
    else prom.resolve(token)
  })
  failedQueue = []
}

// Response interceptor: handle 401 and token refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      if (error.response?.data?.code === 'TOKEN_EXPIRED') {
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({ resolve, reject })
          })
            .then((token) => {
              originalRequest.headers.Authorization = `Bearer ${token}`
              return api(originalRequest)
            })
            .catch((err) => Promise.reject(err))
        }

        originalRequest._retry = true
        isRefreshing = true

        const refreshToken = localStorage.getItem('refresh_token')
        if (!refreshToken) {
          const authStore = useAuthStore()
          authStore.logout()
          router.push('/login')
          return Promise.reject(error)
        }

        try {
          const { data } = await axios.post(
            `${import.meta.env.VITE_API_URL || ''}/api/v1/auth/refresh`,
            { refresh_token: refreshToken }
          )
          localStorage.setItem('access_token', data.access_token)
          localStorage.setItem('refresh_token', data.refresh_token)
          api.defaults.headers.common.Authorization = `Bearer ${data.access_token}`
          processQueue(null, data.access_token)
          originalRequest.headers.Authorization = `Bearer ${data.access_token}`
          return api(originalRequest)
        } catch (refreshError) {
          processQueue(refreshError, null)
          const authStore = useAuthStore()
          authStore.logout()
          router.push('/login')
          return Promise.reject(refreshError)
        } finally {
          isRefreshing = false
        }
      }
    }

    return Promise.reject(error)
  }
)

// --- Auth ---
export const authAPI = {
  login: (data) => api.post('/auth/login', data),
  logout: (refreshToken) => api.post('/auth/logout', { refresh_token: refreshToken }),
  refresh: (refreshToken) => api.post('/auth/refresh', { refresh_token: refreshToken }),
  changePassword: (data) => api.post('/auth/change-password', data),
  setupMFA: () => api.post('/auth/mfa/setup'),
  verifyMFA: (code) => api.post('/auth/mfa/verify', { code }),
}

// --- Dashboard ---
export const dashboardAPI = {
  getStats: () => api.get('/statistics/dashboard'),
  getActiveSessions: (params) => api.get('/sessions/active', { params }),
  getUserSessions: (username) => api.get(`/sessions/user/${username}`),
  getAuthLogs: (params) => api.get('/logs/auth', { params }),
  getAuditLogs: (params) => api.get('/logs/audit', { params }),
}

// --- Admin Users ---
export const adminAPI = {
  list: (params) => api.get('/admin/users', { params }),
  create: (data) => api.post('/admin/users', data),
  update: (id, data) => api.put(`/admin/users/${id}`, data),
  delete: (id) => api.delete(`/admin/users/${id}`),
}

// --- RADIUS Users ---
export const radiusUsersAPI = {
  list: (params) => api.get('/radius/users', { params }),
  get: (id) => api.get(`/radius/users/${id}`),
  create: (data) => api.post('/radius/users', data),
  update: (id, data) => api.put(`/radius/users/${id}`, data),
  delete: (id) => api.delete(`/radius/users/${id}`),
  resetPassword: (id, data) => api.post(`/radius/users/${id}/reset-password`, data),
  suspend: (id) => api.post(`/radius/users/${id}/suspend`),
  activate: (id) => api.post(`/radius/users/${id}/activate`),
  disconnect: (id) => api.post(`/radius/users/${id}/disconnect`),
  getSessions: (id) => api.get(`/radius/users/${id}/sessions`),
  import: (formData) => api.post('/radius/users/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  }),
  export: () => api.get('/radius/users/export', { responseType: 'blob' }),
}

// --- NAS Devices ---
export const nasAPI = {
  list: (params) => api.get('/nas', { params }),
  get: (id) => api.get(`/nas/${id}`),
  create: (data) => api.post('/nas', data),
  update: (id, data) => api.put(`/nas/${id}`, data),
  delete: (id) => api.delete(`/nas/${id}`),
  test: (id) => api.post(`/nas/${id}/test`),
  discover: (data) => api.post('/nas/discover', data),
}

// --- RADIUS Test ---
export const radiusAPI = {
  test: (data) => api.post('/radius/test', data),
}

// --- Settings ---
export const settingsAPI = {
  get: () => api.get('/settings'),
  update: (data) => api.put('/settings', data),
  getBackups: () => api.get('/backups'),
  createBackup: () => api.post('/backup'),
}

// --- Health ---
export const healthAPI = {
  check: () => api.get('/health'),
}

// --- Setup Wizard ---
export const setupAPI = {
  status: () => api.get('/setup/status'),
  complete: (data) => api.post('/setup/complete', data),
}

// ── Tier 1 Pro: Vouchers ──────────────────────────────────────────────────
export const voucherAPI = {
  list: (params) => api.get('/vouchers', { params }),
  generate: (data) => api.post('/vouchers/generate', data),
  batches: () => api.get('/vouchers/batches'),
  disable: (id) => api.post(`/vouchers/${id}/disable`),
  delete: (id) => api.delete(`/vouchers/${id}`),
  export: (params) => api.get('/vouchers/export', { params, responseType: 'blob' }),
}

// ── Tier 1 Pro: Bandwidth Profiles ───────────────────────────────────────
export const bandwidthAPI = {
  list: () => api.get('/bandwidth-profiles'),
  create: (data) => api.post('/bandwidth-profiles', data),
  update: (id, data) => api.put(`/bandwidth-profiles/${id}`, data),
  delete: (id) => api.delete(`/bandwidth-profiles/${id}`),
  applyToUser: (userId, profileId) => api.post(`/radius/users/${userId}/bandwidth`, { profile_id: profileId }),
}

// ── Tier 1 Pro: Reports ───────────────────────────────────────────────────
export const reportsAPI = {
  usage: (params) => api.get('/reports/usage', { params }),
  daily: (params) => api.get('/reports/usage/daily', { params }),
  auth: (params) => api.get('/reports/auth', { params }),
  nas: (params) => api.get('/reports/nas', { params }),
  exportUsage: (params) => api.get('/reports/usage/export', { params, responseType: 'blob' }),
}

export default api
