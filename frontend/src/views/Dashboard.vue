<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Dashboard</h1>
        <p class="text-sm text-gray-500 mt-0.5">Real-time network authentication overview</p>
      </div>
      <button @click="loadStats" class="btn-secondary" :disabled="loading">
        <ArrowPathIcon class="w-4 h-4" :class="{ spinner: loading }" />
        Refresh
      </button>
    </div>

    <!-- Stats grid -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <StatCard
        title="Active Sessions"
        :value="stats.summary?.active_sessions ?? '—'"
        icon="signal"
        color="blue"
        :loading="loading"
      />
      <StatCard
        title="Total Users"
        :value="stats.summary?.total_users ?? '—'"
        icon="users"
        color="purple"
        :loading="loading"
      />
      <StatCard
        title="Active Users"
        :value="stats.summary?.active_users ?? '—'"
        icon="user-check"
        color="green"
        :loading="loading"
      />
      <StatCard
        title="NAS Devices"
        :value="stats.summary?.total_nas ?? '—'"
        icon="server"
        color="orange"
        :loading="loading"
      />
    </div>

    <!-- Charts row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Auth activity chart -->
      <div class="card">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-gray-900">Authentication Activity (24h)</h3>
          <span class="badge badge-blue">{{ stats.summary?.today_auths ?? 0 }} today</span>
        </div>
        <div class="h-48">
          <Bar v-if="chartData.labels?.length" :data="chartData" :options="chartOptions" />
          <div v-else class="h-full flex items-center justify-center text-gray-400 text-sm">
            No data available
          </div>
        </div>
      </div>

      <!-- Top users -->
      <div class="card">
        <h3 class="font-semibold text-gray-900 mb-4">Top Users (7 days)</h3>
        <div v-if="stats.top_users?.length" class="space-y-3">
          <div
            v-for="(user, i) in stats.top_users.slice(0, 6)"
            :key="user.username"
            class="flex items-center gap-3"
          >
            <span class="w-6 text-xs text-gray-400 text-right">{{ i + 1 }}</span>
            <div class="flex-1">
              <div class="flex items-center justify-between mb-0.5">
                <span class="text-sm font-medium text-gray-800">{{ user.username }}</span>
                <span class="text-xs text-gray-500">{{ user.sessions }} sessions</span>
              </div>
              <div class="w-full bg-gray-100 rounded-full h-1.5">
                <div
                  class="bg-blue-500 h-1.5 rounded-full transition-all duration-500"
                  :style="{ width: `${(user.sessions / maxSessions) * 100}%` }"
                ></div>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="h-40 flex items-center justify-center text-gray-400 text-sm">
          No session data available
        </div>
      </div>
    </div>

    <!-- Recent authentications table -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <h3 class="font-semibold text-gray-900">Recent Authentications</h3>
        <router-link to="/monitor" class="text-sm text-blue-600 hover:underline">View all</router-link>
      </div>

      <div v-if="loading" class="h-32 flex items-center justify-center">
        <span class="w-6 h-6 border-2 border-blue-600 border-t-transparent rounded-full spinner"></span>
      </div>

      <div v-else-if="stats.recent_auths?.length" class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>Username</th>
              <th>NAS IP</th>
              <th>Device</th>
              <th>Connected</th>
              <th>Duration</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="auth in stats.recent_auths" :key="auth.session_id || auth.username">
              <td class="font-medium">{{ auth.username }}</td>
              <td class="text-gray-500">{{ auth.nas_ip }}</td>
              <td class="text-gray-500 font-mono text-xs">{{ auth.framed_ip || '—' }}</td>
              <td class="text-gray-500">{{ formatDate(auth.start_time) }}</td>
              <td class="text-gray-500">{{ formatDuration(auth.session_time) }}</td>
              <td>
                <span :class="auth.active ? 'badge-green' : 'badge-gray'" class="badge">
                  {{ auth.active ? 'Active' : 'Ended' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else class="h-24 flex items-center justify-center text-gray-400 text-sm">
        No recent authentication data
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Bar } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend } from 'chart.js'
import { dashboardAPI } from '@/api'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import StatCard from '@/components/dashboard/StatCard.vue'
import { formatDistanceToNow, parseISO, format } from 'date-fns'

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend)

const stats = ref({})
const loading = ref(false)
let refreshTimer

const chartData = computed(() => {
  const data = stats.value.auth_stats_24h || []
  return {
    labels: data.map(d => d.hour.slice(11, 16)),
    datasets: [{
      label: 'Sessions',
      data: data.map(d => d.sessions),
      backgroundColor: 'rgba(59, 130, 246, 0.6)',
      borderColor: 'rgb(59, 130, 246)',
      borderWidth: 1,
      borderRadius: 4,
    }],
  }
})

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: { mode: 'index', intersect: false },
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: { precision: 0 },
      grid: { color: 'rgba(0,0,0,0.05)' },
    },
    x: {
      grid: { display: false },
    },
  },
}

const maxSessions = computed(() => {
  const users = stats.value.top_users || []
  return Math.max(...users.map(u => u.sessions), 1)
})

async function loadStats() {
  loading.value = true
  try {
    const { data } = await dashboardAPI.getStats()
    stats.value = data
  } catch (err) {
    console.error('Failed to load dashboard stats:', err)
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr) {
  if (!dateStr) return '—'
  try {
    return format(parseISO(dateStr), 'MMM d, HH:mm')
  } catch {
    return dateStr
  }
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

onMounted(() => {
  loadStats()
  refreshTimer = setInterval(loadStats, 30000)
})

onUnmounted(() => clearInterval(refreshTimer))
</script>
