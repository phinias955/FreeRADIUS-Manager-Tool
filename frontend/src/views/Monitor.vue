<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Monitoring</h1>
        <p class="text-sm text-gray-500 mt-0.5">Active sessions and authentication logs</p>
      </div>
      <button @click="loadAll" class="btn-secondary" :disabled="loading">
        <ArrowPathIcon class="w-4 h-4" :class="{ spinner: loading }" />
        Refresh
      </button>
    </div>

    <!-- Tabs -->
    <div class="border-b border-gray-200">
      <nav class="flex gap-6">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          class="pb-3 text-sm font-medium border-b-2 transition-colors"
          :class="activeTab === tab.id
            ? 'border-blue-600 text-blue-600'
            : 'border-transparent text-gray-500 hover:text-gray-700'"
        >
          {{ tab.label }}
          <span v-if="tab.count !== undefined"
            class="ml-1.5 px-1.5 py-0.5 text-xs rounded-full"
            :class="activeTab === tab.id ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-600'"
          >
            {{ tab.count }}
          </span>
        </button>
      </nav>
    </div>

    <!-- Active Sessions tab -->
    <div v-if="activeTab === 'sessions'">
      <div class="card p-0 overflow-hidden">
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200">
          <p class="text-sm font-medium text-gray-700">{{ sessions.total || 0 }} active sessions</p>
          <span class="flex items-center gap-1.5 text-xs text-green-600">
            <span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
            Live
          </span>
        </div>
        <div class="table-container rounded-none border-0">
          <table class="table">
            <thead>
              <tr>
                <th>Username</th>
                <th>NAS IP</th>
                <th>Framed IP</th>
                <th>Device MAC</th>
                <th>Connected</th>
                <th>Duration</th>
                <th>Data</th>
                <th>Port</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!sessions.data?.length">
                <td colspan="9" class="text-center text-gray-400 py-12">No active sessions</td>
              </tr>
              <tr v-for="s in sessions.data" :key="s.session_id">
                <td class="font-medium text-gray-900">{{ s.username }}</td>
                <td class="text-gray-500 font-mono text-xs">{{ s.nas_ip }}</td>
                <td class="font-mono text-xs text-blue-600">{{ s.framed_ip || '—' }}</td>
                <td class="font-mono text-xs text-gray-500">{{ formatMAC(s.calling_station) }}</td>
                <td class="text-gray-500 text-xs">{{ formatDate(s.start_time) }}</td>
                <td class="text-gray-500">{{ formatDuration(s.duration_seconds) }}</td>
                <td class="text-gray-500 text-xs">{{ formatBytes(s.input_bytes + s.output_bytes) }}</td>
                <td class="text-gray-400 text-xs">{{ s.nas_port || '—' }}</td>
                <td>
                  <button @click="disconnectSession(s)" class="text-xs px-2 py-1 rounded bg-red-100 text-red-600 hover:bg-red-200">
                    Kick
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Auth Logs tab -->
    <div v-if="activeTab === 'logs'">
      <!-- Filters -->
      <div class="flex gap-3 mb-4">
        <input
          v-model="logSearch"
          type="text"
          class="form-input flex-1"
          placeholder="Filter by username..."
          @input="debouncedLogSearch"
        />
        <button @click="loadLogs" class="btn-secondary px-4">
          <MagnifyingGlassIcon class="w-4 h-4" />
        </button>
      </div>

      <div class="card p-0 overflow-hidden">
        <div class="table-container rounded-none border-0">
          <table class="table">
            <thead>
              <tr>
                <th>Username</th>
                <th>NAS IP</th>
                <th>Device</th>
                <th>Started</th>
                <th>Stopped</th>
                <th>Duration</th>
                <th>Term Cause</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!authLogs.data?.length">
                <td colspan="8" class="text-center text-gray-400 py-12">No auth logs found</td>
              </tr>
              <tr v-for="l in authLogs.data" :key="l.session_id">
                <td class="font-medium">{{ l.username }}</td>
                <td class="font-mono text-xs text-gray-500">{{ l.nas_ip }}</td>
                <td class="font-mono text-xs text-gray-500">{{ formatMAC(l.calling_station) }}</td>
                <td class="text-xs text-gray-500">{{ formatDate(l.start_time) }}</td>
                <td class="text-xs text-gray-500">{{ l.stop_time ? formatDate(l.stop_time) : '—' }}</td>
                <td class="text-gray-500">{{ formatDuration(l.duration) }}</td>
                <td class="text-xs text-gray-400">{{ l.term_cause || '—' }}</td>
                <td>
                  <span :class="l.active ? 'badge-green' : 'badge-gray'" class="badge">
                    {{ l.active ? 'Active' : 'Ended' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Audit Log tab -->
    <div v-if="activeTab === 'audit'">
      <div class="card p-0 overflow-hidden">
        <div class="table-container rounded-none border-0">
          <table class="table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Admin</th>
                <th>Action</th>
                <th>Details</th>
                <th>IP</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!auditLogs.data?.length">
                <td colspan="5" class="text-center text-gray-400 py-12">No audit log entries</td>
              </tr>
              <tr v-for="l in auditLogs.data" :key="l.id">
                <td class="text-xs text-gray-500">{{ formatDate(l.created_at) }}</td>
                <td class="font-medium text-sm">{{ l.username || `User #${l.user_id}` }}</td>
                <td>
                  <span class="font-mono text-xs bg-gray-100 text-gray-700 px-2 py-0.5 rounded">
                    {{ l.action }}
                  </span>
                </td>
                <td class="text-xs text-gray-500 max-w-xs truncate">{{ l.details || '—' }}</td>
                <td class="font-mono text-xs text-gray-400">{{ l.ip_address || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { dashboardAPI, radiusUsersAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { ArrowPathIcon, MagnifyingGlassIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'

const toast = useToast()
const loading = ref(false)
const activeTab = ref('sessions')
const sessions = ref({ data: [], total: 0 })
const authLogs = ref({ data: [] })
const auditLogs = ref({ data: [], total: 0 })
const logSearch = ref('')
let refreshTimer, searchTimer

const tabs = computed(() => [
  { id: 'sessions', label: 'Active Sessions', count: sessions.value.total },
  { id: 'logs', label: 'Auth Logs' },
  { id: 'audit', label: 'Audit Log' },
])

async function loadSessions() {
  try {
    const { data } = await dashboardAPI.getActiveSessions({ limit: 50 })
    sessions.value = data
  } catch { /* silent */ }
}

async function loadLogs() {
  try {
    const { data } = await dashboardAPI.getAuthLogs({ username: logSearch.value, limit: 50 })
    authLogs.value = data
  } catch { /* silent */ }
}

async function loadAuditLogs() {
  try {
    const { data } = await dashboardAPI.getAuditLogs({ limit: 50 })
    auditLogs.value = data
  } catch { /* silent */ }
}

function debouncedLogSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(loadLogs, 400)
}

async function loadAll() {
  loading.value = true
  await Promise.all([loadSessions(), loadLogs(), loadAuditLogs()])
  loading.value = false
}

async function disconnectSession(session) {
  if (!confirm(`Disconnect user "${session.username}"?`)) return
  try {
    // Look up user by username to get their ID, then call disconnect
    const { data: usersData } = await radiusUsersAPI.list({ search: session.username, limit: 5 })
    const user = (usersData.data || []).find(u => u.username === session.username)
    if (!user) { toast.error('User not found'); return }
    await radiusUsersAPI.disconnect(user.id)
    toast.success(`${session.username} disconnected`)
    loadSessions()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Disconnect failed')
  }
}

function formatDate(dateStr) {
  if (!dateStr) return '—'
  try { return format(parseISO(dateStr), 'MMM d, HH:mm:ss') } catch { return dateStr }
}

function formatDuration(seconds) {
  if (!seconds) return '—'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i]
}

function formatMAC(mac) {
  if (!mac) return '—'
  return mac.toUpperCase()
}

onMounted(() => {
  loadAll()
  refreshTimer = setInterval(loadSessions, 15000)
})
onUnmounted(() => clearInterval(refreshTimer))
</script>
