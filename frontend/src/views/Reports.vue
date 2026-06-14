<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Reports</h1>
        <p class="text-sm text-gray-500 mt-0.5">Usage analytics and authentication statistics</p>
      </div>
      <div class="flex gap-2">
        <select v-model="period" @change="loadAll" class="form-input w-32">
          <option value="1d">Today</option>
          <option value="7d">7 Days</option>
          <option value="30d">30 Days</option>
          <option value="90d">90 Days</option>
        </select>
        <button @click="exportUsage" class="btn-secondary">
          <ArrowDownTrayIcon class="w-4 h-4" />
          Export CSV
        </button>
      </div>
    </div>

    <!-- Summary row -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <div class="card p-4">
        <p class="text-xs text-gray-500">Total Sessions</p>
        <p class="text-2xl font-bold text-gray-900 mt-1">{{ summary.sessions }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs text-gray-500">Data Transferred</p>
        <p class="text-2xl font-bold text-blue-600 mt-1">{{ summary.totalGB }} GB</p>
      </div>
      <div class="card p-4">
        <p class="text-xs text-gray-500">Unique Users</p>
        <p class="text-2xl font-bold text-green-600 mt-1">{{ summary.users }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs text-gray-500">Auth Success Rate</p>
        <p class="text-2xl font-bold mt-1" :class="summary.successRate >= 90 ? 'text-green-600' : 'text-yellow-600'">
          {{ summary.successRate }}%
        </p>
      </div>
    </div>

    <!-- Tabs -->
    <div class="border-b border-gray-200">
      <nav class="flex gap-6">
        <button v-for="tab in tabs" :key="tab.id" @click="activeTab = tab.id"
          class="pb-3 text-sm font-medium border-b-2 transition-colors"
          :class="activeTab === tab.id ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'">
          {{ tab.label }}
        </button>
      </nav>
    </div>

    <!-- Daily Traffic Chart -->
    <div v-if="activeTab === 'traffic'" class="card p-5">
      <h3 class="text-sm font-semibold text-gray-700 mb-4">Daily Traffic (GB)</h3>
      <div class="h-56 flex items-end gap-1">
        <template v-if="dailyData.length">
          <div v-for="d in dailyData" :key="d.day"
            class="flex-1 flex flex-col items-center gap-1 group cursor-pointer min-w-0"
            :title="`${d.day}: ${d.total_gb.toFixed(2)} GB, ${d.sessions} sessions`">
            <div class="w-full bg-blue-500 rounded-t transition-all hover:bg-blue-600"
              :style="{ height: barHeight(d.total_gb, maxGB) }"></div>
            <span class="text-xs text-gray-400 truncate w-full text-center hidden sm:block"
              style="font-size:9px">{{ d.day.slice(5) }}</span>
          </div>
        </template>
        <div v-else class="w-full text-center text-gray-400 py-12">No data for this period</div>
      </div>
      <!-- Legend -->
      <div class="flex flex-wrap gap-4 mt-4 pt-4 border-t border-gray-100">
        <div v-for="d in dailyData.slice(-7)" :key="d.day" class="text-xs text-gray-500">
          <span class="font-medium text-gray-700">{{ d.day }}</span>:
          {{ d.total_gb.toFixed(2) }} GB · {{ d.sessions }} sessions
        </div>
      </div>
    </div>

    <!-- Auth Success Chart -->
    <div v-if="activeTab === 'auth'" class="card p-5">
      <h3 class="text-sm font-semibold text-gray-700 mb-4">Authentication Results</h3>
      <div class="h-56 flex items-end gap-2">
        <template v-if="authData.length">
          <div v-for="d in authData" :key="d.day"
            class="flex-1 flex flex-col items-center gap-0.5 min-w-0"
            :title="`${d.day}: ${d.accepted} accept, ${d.rejected} reject`">
            <div class="w-full flex flex-col justify-end" style="height: 200px">
              <div class="w-full bg-red-400 rounded-sm" :style="{ height: barHeight(d.rejected, maxAuth) }"></div>
              <div class="w-full bg-green-500 rounded-t" :style="{ height: barHeight(d.accepted, maxAuth) }"></div>
            </div>
            <span class="text-xs text-gray-400 w-full text-center" style="font-size:9px">{{ d.day.slice(5) }}</span>
          </div>
        </template>
        <div v-else class="w-full text-center text-gray-400 py-12">No auth data for this period</div>
      </div>
      <div class="flex gap-4 mt-3">
        <span class="flex items-center gap-1 text-xs"><span class="w-3 h-3 rounded bg-green-500"></span> Accepted</span>
        <span class="flex items-center gap-1 text-xs"><span class="w-3 h-3 rounded bg-red-400"></span> Rejected</span>
      </div>
    </div>

    <!-- Top Users Table -->
    <div v-if="activeTab === 'users'" class="card p-0 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-200 flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-700">Top Users by Data Usage</h3>
        <span class="text-xs text-gray-400">{{ period }} period</span>
      </div>
      <div class="table-container rounded-none border-0">
        <table class="table">
          <thead>
            <tr>
              <th>#</th>
              <th>Username</th>
              <th>Sessions</th>
              <th>Download</th>
              <th>Upload</th>
              <th>Total</th>
              <th>Avg Session</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!usageData.length">
              <td colspan="7" class="text-center py-12 text-gray-400">No usage data for this period</td>
            </tr>
            <tr v-for="(u, i) in usageData" :key="u.username">
              <td class="text-gray-400 text-sm">{{ i + 1 }}</td>
              <td class="font-medium">{{ u.username }}</td>
              <td class="text-gray-600">{{ u.sessions }}</td>
              <td class="text-gray-600">{{ formatBytes(u.output_bytes) }}</td>
              <td class="text-gray-600">{{ formatBytes(u.input_bytes) }}</td>
              <td class="font-semibold text-blue-600">{{ formatBytes(u.total_bytes) }}</td>
              <td class="text-gray-500 text-sm">{{ formatDuration(u.avg_duration_seconds) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- NAS Stats -->
    <div v-if="activeTab === 'nas'" class="card p-0 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-200">
        <h3 class="text-sm font-semibold text-gray-700">Traffic by NAS Device</h3>
      </div>
      <div class="table-container rounded-none border-0">
        <table class="table">
          <thead>
            <tr>
              <th>NAS IP</th>
              <th>Sessions</th>
              <th>Unique Users</th>
              <th>Total Traffic</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!nasData.length">
              <td colspan="4" class="text-center py-12 text-gray-400">No NAS data for this period</td>
            </tr>
            <tr v-for="n in nasData" :key="n.nas_ip">
              <td class="font-mono text-sm text-gray-700">{{ n.nas_ip }}</td>
              <td class="text-gray-600">{{ n.sessions }}</td>
              <td class="text-gray-600">{{ n.unique_users }}</td>
              <td class="font-semibold text-blue-600">{{ formatBytes(n.total_bytes) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { reportsAPI } from '@/api'
import { ArrowDownTrayIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'

const toast = useToast()
const period = ref('7d')
const activeTab = ref('traffic')
const dailyData = ref([])
const authData = ref([])
const usageData = ref([])
const nasData = ref([])

const tabs = [
  { id: 'traffic', label: 'Daily Traffic' },
  { id: 'auth',    label: 'Auth Success/Fail' },
  { id: 'users',   label: 'Top Users' },
  { id: 'nas',     label: 'Per NAS' },
]

const summary = computed(() => {
  const sessions = usageData.value.reduce((s, u) => s + u.sessions, 0)
  const totalGB = usageData.value.reduce((s, u) => s + u.total_gb, 0).toFixed(2)
  const users = usageData.value.length
  const totalAccept = authData.value.reduce((s, d) => s + d.accepted, 0)
  const totalReject = authData.value.reduce((s, d) => s + d.rejected, 0)
  const successRate = totalAccept + totalReject > 0
    ? ((totalAccept / (totalAccept + totalReject)) * 100).toFixed(1)
    : '—'
  return { sessions, totalGB, users, successRate }
})

const maxGB = computed(() => Math.max(...dailyData.value.map(d => d.total_gb), 1))
const maxAuth = computed(() => Math.max(...authData.value.map(d => d.accepted + d.rejected), 1))

function barHeight(val, max) {
  const pct = Math.max(2, (val / max) * 180)
  return `${pct.toFixed(0)}px`
}

function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024, sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1)
  return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i]
}

function formatDuration(sec) {
  if (!sec) return '—'
  const h = Math.floor(sec / 3600), m = Math.floor((sec % 3600) / 60)
  if (h) return `${h}h ${m}m`
  return `${m}m`
}

async function loadAll() {
  try {
    const [daily, auth, usage, nas] = await Promise.allSettled([
      reportsAPI.daily({ period: period.value }),
      reportsAPI.auth({ period: period.value }),
      reportsAPI.usage({ period: period.value }),
      reportsAPI.nas({ period: period.value }),
    ])
    dailyData.value = daily.status === 'fulfilled' ? daily.value.data.data || [] : []
    authData.value = auth.status === 'fulfilled' ? auth.value.data.data || [] : []
    usageData.value = usage.status === 'fulfilled' ? usage.value.data.data || [] : []
    nasData.value = nas.status === 'fulfilled' ? nas.value.data.data || [] : []
  } catch { /* silent */ }
}

async function exportUsage() {
  try {
    const { data } = await reportsAPI.exportUsage({ period: period.value })
    const url = URL.createObjectURL(new Blob([data]))
    const a = document.createElement('a')
    a.href = url
    a.download = `usage_report_${period.value}_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  } catch { toast.error('Export failed') }
}

onMounted(loadAll)
</script>
