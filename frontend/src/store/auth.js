import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authAPI } from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))
  const accessToken = ref(localStorage.getItem('access_token') || null)
  const refreshToken = ref(localStorage.getItem('refresh_token') || null)

  const isAuthenticated = computed(() => !!accessToken.value && !!user.value)
  const userRole = computed(() => user.value?.role || null)

  const isSuperAdmin = computed(() => userRole.value === 'super_admin')
  const isAdmin = computed(() => ['super_admin', 'admin'].includes(userRole.value))
  const isOperator = computed(() => ['super_admin', 'admin', 'operator'].includes(userRole.value))

  async function login(username, password, mfaCode = '') {
    const { data } = await authAPI.login({ username, password, mfa_code: mfaCode })

    if (data.mfa_required) {
      return { mfaRequired: true }
    }

    accessToken.value = data.access_token
    refreshToken.value = data.refresh_token
    user.value = data.user

    localStorage.setItem('access_token', data.access_token)
    localStorage.setItem('refresh_token', data.refresh_token)
    localStorage.setItem('user', JSON.stringify(data.user))

    return { success: true, user: data.user }
  }

  async function logout() {
    try {
      if (refreshToken.value) {
        await authAPI.logout(refreshToken.value)
      }
    } catch {
      // Ignore logout errors
    } finally {
      clearAuth()
    }
  }

  function clearAuth() {
    user.value = null
    accessToken.value = null
    refreshToken.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('user')
  }

  function hasRole(...roles) {
    return roles.includes(userRole.value)
  }

  function setUser(updatedUser) {
    user.value = updatedUser
    localStorage.setItem('user', JSON.stringify(updatedUser))
  }

  return {
    user,
    accessToken,
    refreshToken,
    isAuthenticated,
    userRole,
    isSuperAdmin,
    isAdmin,
    isOperator,
    login,
    logout,
    clearAuth,
    hasRole,
    setUser,
  }
})
