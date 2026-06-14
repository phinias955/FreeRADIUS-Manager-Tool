<template>
  <div class="min-h-screen bg-gradient-to-br from-blue-900 via-blue-800 to-indigo-900 flex items-center justify-center p-4">
    <!-- Background pattern -->
    <div class="absolute inset-0 opacity-10">
      <div class="absolute inset-0" style="background-image: url(&quot;data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='1'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E&quot;);"></div>
    </div>

    <div class="relative w-full max-w-md">
      <!-- Logo -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 bg-white/10 backdrop-blur-sm rounded-2xl mb-4 border border-white/20">
          <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
          </svg>
        </div>
        <h1 class="text-3xl font-bold text-white">FreeRADIUS Manager</h1>
        <p class="text-blue-200 mt-2">Enterprise Network Authentication Platform</p>
      </div>

      <!-- Login card -->
      <div class="bg-white rounded-2xl shadow-2xl p-8">
        <h2 class="text-xl font-bold text-gray-900 mb-6">Sign in to your account</h2>

        <!-- MFA step -->
        <template v-if="mfaRequired">
          <form @submit.prevent="submitMFA" class="space-y-5">
            <div class="p-4 bg-blue-50 rounded-lg border border-blue-200">
              <p class="text-sm text-blue-700 font-medium">Two-Factor Authentication Required</p>
              <p class="text-sm text-blue-600 mt-1">Enter the 6-digit code from your authenticator app.</p>
            </div>

            <div>
              <label class="form-label">Authentication Code</label>
              <input
                v-model="mfaCode"
                type="text"
                inputmode="numeric"
                pattern="[0-9]{6}"
                maxlength="6"
                placeholder="000000"
                class="form-input text-center text-2xl tracking-widest font-mono"
                autofocus
                required
              />
            </div>

            <button type="submit" class="btn-primary w-full justify-center py-3" :disabled="loading">
              <span v-if="loading" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Verify Code
            </button>

            <button type="button" class="btn-secondary w-full justify-center" @click="mfaRequired = false">
              Back to Login
            </button>
          </form>
        </template>

        <!-- Standard login -->
        <template v-else>
          <form @submit.prevent="handleLogin" class="space-y-5">
            <div v-if="errorMsg" class="p-3 bg-red-50 border border-red-200 rounded-lg">
              <p class="text-sm text-red-600">{{ errorMsg }}</p>
            </div>

            <div>
              <label class="form-label">Username</label>
              <input
                v-model="form.username"
                type="text"
                class="form-input"
                placeholder="Enter username"
                autocomplete="username"
                required
                autofocus
              />
            </div>

            <div>
              <label class="form-label">Password</label>
              <div class="relative">
                <input
                  v-model="form.password"
                  :type="showPassword ? 'text' : 'password'"
                  class="form-input pr-10"
                  placeholder="Enter password"
                  autocomplete="current-password"
                  required
                />
                <button
                  type="button"
                  class="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 hover:text-gray-600"
                  @click="showPassword = !showPassword"
                >
                  <EyeIcon v-if="!showPassword" class="w-4 h-4" />
                  <EyeSlashIcon v-else class="w-4 h-4" />
                </button>
              </div>
            </div>

            <button
              type="submit"
              class="btn-primary w-full justify-center py-3 text-base"
              :disabled="loading"
            >
              <span v-if="loading" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              <span>{{ loading ? 'Signing in...' : 'Sign In' }}</span>
            </button>
          </form>
        </template>
      </div>

      <p class="text-center text-blue-200/60 text-xs mt-6">
        Secured by FreeRADIUS &amp; JWT authentication
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { useToast } from 'vue-toastification'
import { EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const toast = useToast()

const form = ref({ username: '', password: '' })
const mfaCode = ref('')
const mfaRequired = ref(false)
const loading = ref(false)
const showPassword = ref(false)
const errorMsg = ref('')

async function handleLogin() {
  errorMsg.value = ''
  loading.value = true

  try {
    const result = await authStore.login(form.value.username, form.value.password)

    if (result.mfaRequired) {
      mfaRequired.value = true
      return
    }

    const redirect = route.query.redirect || '/dashboard'
    router.push(redirect)
    toast.success(`Welcome back, ${result.user.full_name || result.user.username}!`)
  } catch (err) {
    errorMsg.value = err.response?.data?.error || 'Login failed. Please check your credentials.'
  } finally {
    loading.value = false
  }
}

async function submitMFA() {
  loading.value = true
  try {
    const result = await authStore.login(form.value.username, form.value.password, mfaCode.value)
    const redirect = route.query.redirect || '/dashboard'
    router.push(redirect)
    toast.success('Authenticated successfully')
  } catch (err) {
    toast.error(err.response?.data?.error || 'Invalid MFA code')
  } finally {
    loading.value = false
  }
}
</script>
