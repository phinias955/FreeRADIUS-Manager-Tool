<template>
  <div class="min-h-screen bg-gradient-to-br from-blue-900 via-blue-800 to-indigo-900 flex items-center justify-center p-4">

    <!-- Login form -->
    <div v-if="!session" class="w-full max-w-sm">
      <div class="text-center mb-8">
        <div class="w-16 h-16 bg-white/10 rounded-2xl flex items-center justify-center mx-auto mb-4">
          <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
          </svg>
        </div>
        <h1 class="text-2xl font-bold text-white">My Account Portal</h1>
        <p class="text-blue-200 text-sm mt-1">Check your internet usage and plan details</p>
      </div>

      <div class="bg-white rounded-2xl shadow-2xl p-8">
        <form @submit.prevent="login" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Username</label>
            <input v-model="creds.username" type="text" required autocomplete="username"
              class="w-full border border-gray-300 rounded-xl px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Your internet username" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input v-model="creds.password" type="password" required autocomplete="current-password"
              class="w-full border border-gray-300 rounded-xl px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Your password" />
          </div>
          <p v-if="loginError" class="text-red-500 text-sm text-center">{{ loginError }}</p>
          <button type="submit" :disabled="logging"
            class="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold rounded-xl py-3 transition-colors flex items-center justify-center gap-2">
            <span v-if="logging" class="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
            {{ logging ? 'Signing in…' : 'Sign In' }}
          </button>
        </form>
      </div>
    </div>

    <!-- Dashboard -->
    <div v-else class="w-full max-w-3xl space-y-5">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-white">Hello, {{ data?.user?.username }}</h1>
          <p class="text-blue-200 text-sm mt-0.5">
            <span class="inline-flex items-center gap-1.5">
              <span class="w-2 h-2 rounded-full" :class="data?.user?.status === 'active' ? 'bg-green-400' : 'bg-red-400'"></span>
              Account {{ data?.user?.status }}
              <span v-if="data?.user?.account_expiry">— expires {{ data.user.account_expiry }}</span>
            </span>
          </p>
        </div>
        <button @click="logout" class="text-blue-200 hover:text-white text-sm flex items-center gap-1">
          <ArrowRightOnRectangleIcon class="w-4 h-4" />
          Sign Out
        </button>
      </div>

      <!-- Plan card -->
      <div class="bg-white/10 backdrop-blur rounded-2xl p-5 text-white">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-blue-200 text-xs uppercase tracking-wider">Current Plan</p>
            <p class="text-2xl font-bold mt-1">{{ data?.user?.plan_name || 'No Plan' }}</p>
            <p v-if="data?.user?.plan_price > 0" class="text-blue-200 text-sm mt-1">
              {{ data.user.plan_currency }} {{ data.user.plan_price }} / {{ data.user.validity_days }}d
            </p>
          </div>
          <div v-if="data?.assigned_ip" class="text-right">
            <p class="text-blue-200 text-xs">Your IP</p>
            <p class="font-mono font-bold text-lg">{{ data.assigned_ip }}</p>
          </div>
        </div>

        <!-- Data usage bar -->
        <div v-if="data?.usage" class="mt-5">
          <div class="flex justify-between text-sm mb-2">
            <span>{{ formatMB(data.usage.total_mb) }} used</span>
            <span v-if="data.user.data_limit_mb">{{ formatMB(data.user.data_limit_mb) }} total</span>
            <span v-else>Unlimited</span>
          </div>
          <div v-if="data.user.data_limit_mb" class="w-full bg-white/20 rounded-full h-3">
            <div class="h-3 rounded-full transition-all"
              :class="data.usage.used_pct > 90 ? 'bg-red-400' : data.usage.used_pct > 70 ? 'bg-yellow-400' : 'bg-green-400'"
              :style="{ width: data.usage.used_pct + '%' }">
            </div>
          </div>
          <div class="flex gap-6 mt-3 text-sm text-blue-100">
            <span>↑ {{ formatMB(data.usage.upload_mb) }} upload</span>
            <span>↓ {{ formatMB(data.usage.download_mb) }} download</span>
            <span>{{ data.usage.session_count }} sessions</span>
          </div>
        </div>
      </div>

      <!-- Active sessions -->
      <div v-if="data?.active_sessions?.length" class="bg-white rounded-2xl shadow p-5">
        <h3 class="font-semibold text-gray-900 mb-3 flex items-center gap-2">
          <span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
          Active Sessions ({{ data.active_sessions.length }})
        </h3>
        <div class="space-y-2">
          <div v-for="(s, i) in data.active_sessions" :key="i"
            class="flex items-center justify-between bg-green-50 rounded-xl p-3 text-sm">
            <div>
              <p class="font-medium text-gray-900">{{ s.nasname || 'Unknown NAS' }}</p>
              <p class="text-xs text-gray-500 mt-0.5">Since {{ s.start_time }}</p>
            </div>
            <div class="text-right text-xs text-gray-600">
              <p>{{ formatDuration(s.duration_seconds) }}</p>
              <p>↑{{ formatMB(s.input_mb) }} ↓{{ formatMB(s.output_mb) }}</p>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="bg-white/10 backdrop-blur rounded-2xl p-4 text-center text-blue-200 text-sm">
        No active sessions right now
      </div>

      <!-- Session history -->
      <div v-if="data?.session_history?.length" class="bg-white rounded-2xl shadow p-5">
        <h3 class="font-semibold text-gray-900 mb-3">Recent Sessions</h3>
        <table class="w-full text-sm">
          <thead>
            <tr class="text-gray-400 text-xs border-b border-gray-100">
              <th class="text-left py-1.5">Started</th>
              <th class="text-left">Ended</th>
              <th class="text-right">Data</th>
              <th class="text-right">Duration</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(s, i) in data.session_history" :key="i" class="border-b border-gray-50">
              <td class="py-1.5 text-gray-700">{{ s.start_time }}</td>
              <td class="text-gray-500">{{ s.stop_time }}</td>
              <td class="text-right font-medium">{{ formatMB(s.total_mb) }}</td>
              <td class="text-right text-gray-500">{{ formatDuration(s.duration_seconds) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ArrowRightOnRectangleIcon } from '@heroicons/vue/24/outline'

const SESSION_KEY = 'portal_token'
const API = '/api/v1/portal'

const session = ref(null)
const data = ref(null)
const logging = ref(false)
const loginError = ref('')
const creds = reactive({ username: '', password: '' })

function formatMB(mb) {
  if (!mb) return '0 MB'
  if (mb >= 1024) return (mb / 1024).toFixed(2) + ' GB'
  return mb.toFixed(1) + ' MB'
}

function formatDuration(secs) {
  if (!secs) return '0m'
  const h = Math.floor(secs / 3600)
  const m = Math.floor((secs % 3600) / 60)
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

async function login() {
  logging.value = true
  loginError.value = ''
  try {
    const { data: resp } = await axios.post(API + '/login', creds)
    session.value = resp.token
    localStorage.setItem(SESSION_KEY, resp.token)
    await loadDashboard()
  } catch (err) {
    loginError.value = err.response?.data?.error || 'Login failed'
  } finally { logging.value = false }
}

async function loadDashboard() {
  try {
    const { data: resp } = await axios.get(API + '/dashboard', {
      headers: { 'X-Portal-Token': session.value }
    })
    data.value = resp
  } catch (err) {
    if (err.response?.status === 401) {
      session.value = null
      localStorage.removeItem(SESSION_KEY)
    }
  }
}

async function logout() {
  await axios.post(API + '/logout', {}, {
    headers: { 'X-Portal-Token': session.value }
  }).catch(() => {})
  session.value = null
  data.value = null
  localStorage.removeItem(SESSION_KEY)
}

onMounted(async () => {
  const saved = localStorage.getItem(SESSION_KEY)
  if (saved) {
    session.value = saved
    await loadDashboard()
  }
})
</script>
