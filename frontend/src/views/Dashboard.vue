<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Dashboard</h1>
        <p class="text-sm text-gray-500 mt-0.5">Enterprise RADIUS operations center</p>
      </div>
      <div class="flex items-center gap-3">
        <div class="hidden sm:flex items-center gap-2 text-xs text-gray-500">
          <span class="relative flex h-2 w-2">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
          </span>
          <span>Auto-refresh · {{ refreshInterval / 1000 }}s</span>
          <span v-if="lastUpdated">· {{ lastUpdatedLabel }}</span>
        </div>
        <select v-model="refreshInterval" class="form-select w-auto text-xs py-1.5">
          <option :value="10000">10s</option>
          <option :value="15000">15s</option>
          <option :value="30000">30s</option>
          <option :value="60000">60s</option>
        </select>
        <button @click="loadStats(true)" class="btn-secondary" :disabled="initialLoading">
          <ArrowPathIcon class="w-4 h-4" :class="{ spinner: refreshing }" />
          Refresh
        </button>
      </div>
    </div>

    <HealthBanner :health="stats.system_health" />
    <LivePulseBar :live="stats.live" :summary="stats.summary" />
    <QuickActions />

    <!-- Primary stats -->
    <div class="grid grid-cols-2 lg:grid-cols-4 xl:grid-cols-8 gap-3">
      <StatCard title="Active Sessions" :value="stats.summary?.active_sessions ?? 0" icon="signal" color="blue" :loading="initialLoading" />
      <StatCard title="Auth Today" :value="stats.summary?.today_auths ?? 0" :subtitle="`${stats.summary?.today_accepts ?? 0} accepted`" :trend="stats.summary?.yesterday_auths ? stats.summary?.auth_delta_pct : undefined" icon="bolt" color="indigo" :loading="initialLoading" />
      <StatCard title="Success Rate" :value="successRateLabel" icon="check" color="green" :loading="initialLoading" />
      <StatCard title="Rejections" :value="stats.summary?.today_rejects ?? 0" icon="x" color="red" :loading="initialLoading" />
      <StatCard title="Total Users" :value="stats.summary?.total_users ?? 0" icon="users" color="purple" :loading="initialLoading" />
      <StatCard title="Active Users" :value="stats.summary?.active_users ?? 0" icon="user-check" color="teal" :loading="initialLoading" />
      <StatCard title="NAS Online" :value="`${stats.summary?.nas_up ?? 0}/${stats.summary?.total_nas ?? 0}`" icon="server" color="orange" :loading="initialLoading" />
      <StatCard title="Traffic Today" :value="formatBytes(stats.summary?.traffic_today)" icon="chart" color="cyan" :loading="initialLoading" />
    </div>

    <!-- Pro panels: NAS health · Security · User/Voucher breakdown -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- NAS Health -->
      <div class="card">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h3 class="font-semibold text-gray-900">NAS Health</h3>
            <p class="text-xs text-gray-500">Real-time device reachability</p>
          </div>
          <router-link to="/network-map" class="text-xs text-blue-600 hover:underline">Map →</router-link>
        </div>
        <div v-if="stats.nas_health?.length" class="space-y-2">
          <div
            v-for="nas in stats.nas_health"
            :key="nas.id"
            class="flex items-center gap-3 p-2.5 rounded-lg border border-gray-100 hover:bg-gray-50"
          >
            <span class="w-2.5 h-2.5 rounded-full flex-shrink-0" :class="nasDot(nas.ping_status)"></span>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-gray-900 truncate">{{ nas.shortname || nas.nasname }}</p>
              <p class="text-xs text-gray-400 font-mono truncate">{{ nas.nasname }}</p>
            </div>
            <div class="text-right flex-shrink-0">
              <p class="text-xs font-semibold capitalize" :class="nasStatusColor(nas.ping_status)">{{ nas.ping_status || 'unknown' }}</p>
              <p v-if="nas.ping_status === 'up'" class="text-[10px] text-gray-400">{{ nas.ping_latency_ms?.toFixed(0) }}ms</p>
            </div>
          </div>
        </div>
        <EmptyChart v-else message="No NAS devices configured" />
      </div>

      <!-- Security Center -->
      <div class="card">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h3 class="font-semibold text-gray-900">Security Center</h3>
            <p class="text-xs text-gray-500">Threats & alerts overview</p>
          </div>
          <router-link to="/security" class="text-xs text-blue-600 hover:underline">Open →</router-link>
        </div>
        <div class="grid grid-cols-2 gap-2 mb-4">
          <div class="p-3 rounded-lg bg-red-50 border border-red-100 text-center">
            <p class="text-2xl font-bold text-red-700">{{ stats.summary?.critical_alerts ?? 0 }}</p>
            <p class="text-[10px] text-red-600 uppercase tracking-wide">Critical Alerts</p>
          </div>
          <div class="p-3 rounded-lg bg-orange-50 border border-orange-100 text-center">
            <p class="text-2xl font-bold text-orange-700">{{ stats.summary?.blocked_ips ?? 0 }}</p>
            <p class="text-[10px] text-orange-600 uppercase tracking-wide">Blocked IPs</p>
          </div>
          <div class="p-3 rounded-lg bg-purple-50 border border-purple-100 text-center">
            <p class="text-2xl font-bold text-purple-700">{{ stats.summary?.honeypot_today ?? 0 }}</p>
            <p class="text-[10px] text-purple-600 uppercase tracking-wide">Honeypot Probes</p>
          </div>
          <div class="p-3 rounded-lg bg-amber-50 border border-amber-100 text-center">
            <p class="text-2xl font-bold text-amber-700">{{ stats.summary?.unread_alerts ?? 0 }}</p>
            <p class="text-[10px] text-amber-600 uppercase tracking-wide">Unread Alerts</p>
          </div>
        </div>
        <div v-if="stats.recent_alerts?.length" class="space-y-2 max-h-36 overflow-y-auto">
          <div v-for="a in stats.recent_alerts.slice(0, 4)" :key="a.id" class="flex items-center gap-2 text-xs">
            <span class="badge flex-shrink-0" :class="severityBadge(a.severity)">{{ a.severity }}</span>
            <span class="text-gray-600 truncate flex-1">{{ formatAlertType(a.alert_type) }}</span>
            <span class="text-gray-400 flex-shrink-0">{{ formatRelative(a.created_at) }}</span>
          </div>
        </div>
        <p v-else class="text-xs text-gray-400 text-center py-2">No security alerts recorded</p>
      </div>

      <!-- User & Voucher breakdown -->
      <div class="card">
        <div class="mb-4">
          <h3 class="font-semibold text-gray-900">Account Overview</h3>
          <p class="text-xs text-gray-500">User status & voucher inventory</p>
        </div>
        <div class="h-40 flex items-center justify-center">
          <Doughnut v-if="userStatusHasData" :data="userStatusChart" :options="doughnutOptions" />
          <EmptyChart v-else message="No user data" />
        </div>
        <div class="grid grid-cols-3 gap-2 mt-4 pt-4 border-t border-gray-100">
          <div class="text-center">
            <p class="text-lg font-bold text-teal-600">{{ stats.summary?.vouchers_active ?? 0 }}</p>
            <p class="text-[10px] text-gray-500 uppercase">Active Vouchers</p>
          </div>
          <div class="text-center">
            <p class="text-lg font-bold text-blue-600">{{ stats.summary?.vouchers_used ?? 0 }}</p>
            <p class="text-[10px] text-gray-500 uppercase">Redeemed</p>
          </div>
          <div class="text-center">
            <p class="text-lg font-bold text-gray-600">{{ stats.summary?.vouchers_total ?? 0 }}</p>
            <p class="text-[10px] text-gray-500 uppercase">Total</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Live active sessions -->
    <div v-if="stats.live_sessions?.length" class="card">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h3 class="font-semibold text-gray-900">Live Active Sessions</h3>
          <p class="text-xs text-gray-500">{{ stats.summary?.active_sessions }} session(s) in progress</p>
        </div>
        <router-link to="/monitor" class="text-sm text-blue-600 hover:underline">Manage →</router-link>
      </div>
      <div class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>Username</th>
              <th>NAS IP</th>
              <th>Framed IP</th>
              <th>Device</th>
              <th>Duration</th>
              <th>Data</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in stats.live_sessions" :key="s.session_id">
              <td class="font-medium">{{ s.username }}</td>
              <td class="font-mono text-xs text-gray-500">{{ s.nas_ip }}</td>
              <td class="font-mono text-xs text-blue-600">{{ s.framed_ip || '—' }}</td>
              <td class="font-mono text-xs text-gray-500">{{ formatMAC(s.calling_station) }}</td>
              <td class="text-gray-500 text-xs">{{ formatDuration(s.duration_seconds) }}</td>
              <td class="text-gray-500 text-xs">{{ formatBytes((s.input_bytes || 0) + (s.output_bytes || 0)) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Charts row 1 -->
    <div class="grid grid-cols-1 xl:grid-cols-3 gap-6">
      <div class="card xl:col-span-2">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h3 class="font-semibold text-gray-900">Authentication Activity</h3>
            <p class="text-xs text-gray-500 mt-0.5">Accept vs reject · last 24 hours</p>
          </div>
          <span class="badge badge-blue">{{ stats.summary?.today_auths ?? 0 }} today</span>
        </div>
        <div class="h-64">
          <Bar v-if="authHourlyChart.labels.length" :data="authHourlyChart" :options="barChartOptions" />
          <EmptyChart v-else message="No authentication data in the last 24 hours" />
        </div>
      </div>

      <div class="card">
        <div class="mb-4">
          <h3 class="font-semibold text-gray-900">Today's Outcome</h3>
          <p class="text-xs text-gray-500 mt-0.5">Accept vs reject distribution</p>
        </div>
        <div class="h-64 flex items-center justify-center">
          <Doughnut v-if="outcomeChartHasData" :data="outcomeChart" :options="doughnutOptions" />
          <EmptyChart v-else message="No auth attempts today yet" />
        </div>
      </div>
    </div>

    <!-- Charts row 2 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div class="card">
        <div class="mb-4">
          <h3 class="font-semibold text-gray-900">7-Day Auth Trend</h3>
          <p class="text-xs text-gray-500 mt-0.5">Daily accepted and rejected attempts</p>
        </div>
        <div class="h-64">
          <Line v-if="authDailyChart.labels.length" :data="authDailyChart" :options="lineChartOptions" />
          <EmptyChart v-else message="No weekly authentication data" />
        </div>
      </div>

      <div class="card">
        <div class="mb-4">
          <h3 class="font-semibold text-gray-900">Auth Failures (24h)</h3>
          <p class="text-xs text-gray-500 mt-0.5">Rejected attempts per hour — security monitoring</p>
        </div>
        <div class="h-64">
          <Line v-if="failTrendChart.labels.length" :data="failTrendChart" :options="failChartOptions" />
          <EmptyChart v-else message="No failed authentication attempts" />
        </div>
      </div>
    </div>

    <!-- Charts row 3: traffic + network -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div class="card">
        <div class="mb-4">
          <h3 class="font-semibold text-gray-900">Network Traffic</h3>
          <p class="text-xs text-gray-500 mt-0.5">Data transferred · last 24 hours ({{ formatBytes(stats.summary?.traffic_today) }} today · {{ formatBytes(stats.summary?.traffic_7d) }} / 7d)</p>
        </div>
        <div class="h-64">
          <Line v-if="trafficChart.labels.length" :data="trafficChart" :options="trafficChartOptions" />
          <EmptyChart v-else message="No accounting traffic yet — NAS must send RADIUS accounting on port 1813" />
        </div>
      </div>

      <div class="card">
        <div class="mb-4">
          <h3 class="font-semibold text-gray-900">NAS Activity</h3>
          <p class="text-xs text-gray-500 mt-0.5">Authentication attempts by NAS · 7 days</p>
        </div>
        <div class="h-64">
          <Bar v-if="nasChart.labels.length" :data="nasChart" :options="horizontalBarOptions" />
          <EmptyChart v-else message="No NAS authentication activity recorded" />
        </div>
      </div>
    </div>

    <!-- Charts row 4: top users -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div class="card">
        <div class="mb-4">
          <h3 class="font-semibold text-gray-900">Top Users</h3>
          <p class="text-xs text-gray-500 mt-0.5">Most successful authentications · 7 days</p>
        </div>
        <div class="h-56">
          <Bar v-if="topUsersChart.labels.length" :data="topUsersChart" :options="horizontalBarOptions" />
          <EmptyChart v-else message="No user activity in the last 7 days" />
        </div>
      </div>

      <!-- Admin activity -->
      <div class="card">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h3 class="font-semibold text-gray-900">Admin Activity</h3>
            <p class="text-xs text-gray-500">Recent management actions</p>
          </div>
          <router-link to="/monitor" class="text-xs text-blue-600 hover:underline">Audit log →</router-link>
        </div>
        <div v-if="stats.recent_audit?.length" class="space-y-3">
          <div v-for="a in stats.recent_audit" :key="a.id" class="flex items-start gap-3">
            <div class="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center flex-shrink-0">
              <span class="text-xs font-bold text-gray-500">{{ (a.username || 'S')[0].toUpperCase() }}</span>
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm text-gray-800">
                <span class="font-medium">{{ a.username || 'System' }}</span>
                <span class="font-mono text-xs bg-gray-100 px-1.5 py-0.5 rounded ml-1">{{ a.action }}</span>
              </p>
              <p class="text-xs text-gray-400 truncate mt-0.5">{{ a.details || '—' }}</p>
            </div>
            <span class="text-[10px] text-gray-400 flex-shrink-0">{{ formatRelative(a.created_at) }}</span>
          </div>
        </div>
        <EmptyChart v-else message="No admin activity recorded yet" />
      </div>
    </div>

    <!-- Recent authentications -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h3 class="font-semibold text-gray-900">Recent Authentications</h3>
          <p class="text-xs text-gray-500 mt-0.5">Latest RADIUS auth attempts</p>
        </div>
        <router-link to="/monitor" class="text-sm text-blue-600 hover:underline">View all →</router-link>
      </div>

      <div v-if="initialLoading" class="h-32 flex items-center justify-center">
        <span class="w-6 h-6 border-2 border-blue-600 border-t-transparent rounded-full spinner"></span>
      </div>

      <div v-else-if="stats.recent_auths?.length" class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>Username</th>
              <th>NAS IP</th>
              <th>Device</th>
              <th>Time</th>
              <th>Result</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="auth in stats.recent_auths" :key="auth.id">
              <td class="font-medium">{{ auth.username }}</td>
              <td class="text-gray-500 font-mono text-xs">{{ auth.nas_ip || '—' }}</td>
              <td class="text-gray-500 font-mono text-xs">{{ formatMAC(auth.calling_station) }}</td>
              <td class="text-gray-500 text-xs">{{ formatDate(auth.auth_time) }}</td>
              <td>
                <span :class="auth.accepted ? 'badge-green' : 'badge-red'" class="badge">
                  {{ auth.accepted ? 'Accept' : 'Reject' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else class="h-24 flex items-center justify-center text-gray-400 text-sm">
        No recent authentication data — run a test from Security → RADIUS Simulator
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { Bar, Line, Doughnut } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  LineElement,
  PointElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import { dashboardAPI } from '@/api'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import StatCard from '@/components/dashboard/StatCard.vue'
import EmptyChart from '@/components/dashboard/EmptyChart.vue'
import HealthBanner from '@/components/dashboard/HealthBanner.vue'
import LivePulseBar from '@/components/dashboard/LivePulseBar.vue'
import QuickActions from '@/components/dashboard/QuickActions.vue'
import { formatDistanceToNow, parseISO, format } from 'date-fns'

ChartJS.register(
  CategoryScale, LinearScale, BarElement, LineElement,
  PointElement, ArcElement, Title, Tooltip, Legend, Filler,
)

const stats = ref({})
const initialLoading = ref(true)
const refreshing = ref(false)
const lastUpdated = ref(null)
const refreshInterval = ref(15000)
let refreshTimer

const successRateLabel = computed(() => {
  const rate = stats.value.summary?.auth_success_rate
  if (rate == null || Number.isNaN(rate)) return '—'
  return `${rate.toFixed(1)}%`
})

const lastUpdatedLabel = computed(() => {
  if (!lastUpdated.value) return ''
  return formatDistanceToNow(lastUpdated.value, { addSuffix: true })
})

const outcomeChartHasData = computed(() => {
  const accepts = stats.value.summary?.today_accepts ?? 0
  const rejects = stats.value.summary?.today_rejects ?? 0
  return accepts + rejects > 0
})

const userStatusHasData = computed(() => {
  const u = stats.value.user_status || {}
  return (u.active || 0) + (u.suspended || 0) + (u.expired || 0) > 0
})

const userStatusChart = computed(() => {
  const u = stats.value.user_status || {}
  return {
    labels: ['Active', 'Suspended', 'Expired'],
    datasets: [{
      data: [u.active || 0, u.suspended || 0, u.expired || 0],
      backgroundColor: ['rgba(20, 184, 166, 0.85)', 'rgba(239, 68, 68, 0.85)', 'rgba(156, 163, 175, 0.85)'],
      borderColor: ['rgb(13, 148, 136)', 'rgb(220, 38, 38)', 'rgb(107, 114, 128)'],
      borderWidth: 2,
      hoverOffset: 4,
    }],
  }
})

const failTrendChart = computed(() => {
  const data = stats.value.fail_trend_24h || []
  return {
    labels: data.map(d => d.hour?.slice(11, 16) || ''),
    datasets: [{
      label: 'Failed auths',
      data: data.map(d => d.fails),
      borderColor: 'rgb(239, 68, 68)',
      backgroundColor: 'rgba(239, 68, 68, 0.12)',
      fill: true,
      tension: 0.35,
      pointRadius: 2,
    }],
  }
})

const authHourlyChart = computed(() => {
  const data = stats.value.auth_hourly_24h || []
  return {
    labels: data.map(d => d.hour?.slice(11, 16) || ''),
    datasets: [
      {
        label: 'Accepted',
        data: data.map(d => d.accepted),
        backgroundColor: 'rgba(34, 197, 94, 0.75)',
        borderColor: 'rgb(22, 163, 74)',
        borderWidth: 1,
        borderRadius: 4,
      },
      {
        label: 'Rejected',
        data: data.map(d => d.rejected),
        backgroundColor: 'rgba(239, 68, 68, 0.75)',
        borderColor: 'rgb(220, 38, 38)',
        borderWidth: 1,
        borderRadius: 4,
      },
    ],
  }
})

const authDailyChart = computed(() => {
  const data = stats.value.auth_daily_7d || []
  return {
    labels: data.map(d => {
      try { return format(parseISO(d.day), 'EEE d') } catch { return d.day }
    }),
    datasets: [
      {
        label: 'Accepted',
        data: data.map(d => d.accepted),
        borderColor: 'rgb(34, 197, 94)',
        backgroundColor: 'rgba(34, 197, 94, 0.12)',
        fill: true,
        tension: 0.35,
        pointRadius: 3,
      },
      {
        label: 'Rejected',
        data: data.map(d => d.rejected),
        borderColor: 'rgb(239, 68, 68)',
        backgroundColor: 'rgba(239, 68, 68, 0.08)',
        fill: true,
        tension: 0.35,
        pointRadius: 3,
      },
    ],
  }
})

const trafficChart = computed(() => {
  const data = stats.value.traffic_hourly_24h || []
  const hasTraffic = data.some(d => d.bytes > 0)
  if (!hasTraffic) return { labels: [], datasets: [] }
  return {
    labels: data.map(d => d.hour?.slice(11, 16) || ''),
    datasets: [{
      label: 'Traffic (MB)',
      data: data.map(d => +(d.bytes / (1024 * 1024)).toFixed(2)),
      borderColor: 'rgb(59, 130, 246)',
      backgroundColor: 'rgba(59, 130, 246, 0.15)',
      fill: true,
      tension: 0.35,
      pointRadius: 2,
    }],
  }
})

const outcomeChart = computed(() => ({
  labels: ['Accepted', 'Rejected'],
  datasets: [{
    data: [
      stats.value.summary?.today_accepts ?? 0,
      stats.value.summary?.today_rejects ?? 0,
    ],
    backgroundColor: ['rgba(34, 197, 94, 0.85)', 'rgba(239, 68, 68, 0.85)'],
    borderColor: ['rgb(22, 163, 74)', 'rgb(220, 38, 38)'],
    borderWidth: 2,
    hoverOffset: 6,
  }],
}))

const nasChart = computed(() => {
  const data = stats.value.nas_stats_7d || []
  return {
    labels: data.map(d => d.nas_ip?.replace('/32', '') || 'Unknown'),
    datasets: [{
      label: 'Auth attempts',
      data: data.map(d => d.auths),
      backgroundColor: 'rgba(249, 115, 22, 0.75)',
      borderColor: 'rgb(234, 88, 12)',
      borderWidth: 1,
      borderRadius: 4,
    }],
  }
})

const topUsersChart = computed(() => {
  const data = stats.value.top_users || []
  return {
    labels: data.map(d => d.username),
    datasets: [{
      label: 'Successful auths',
      data: data.map(d => d.sessions),
      backgroundColor: 'rgba(139, 92, 246, 0.75)',
      borderColor: 'rgb(124, 58, 237)',
      borderWidth: 1,
      borderRadius: 4,
    }],
  }
})

const baseChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index', intersect: false },
}

const barChartOptions = {
  ...baseChartOptions,
  plugins: {
    legend: { position: 'top', labels: { boxWidth: 12, font: { size: 11 } } },
    tooltip: { mode: 'index', intersect: false },
  },
  scales: {
    y: { beginAtZero: true, ticks: { precision: 0 }, grid: { color: 'rgba(0,0,0,0.05)' } },
    x: { grid: { display: false } },
  },
}

const lineChartOptions = {
  ...baseChartOptions,
  plugins: {
    legend: { position: 'top', labels: { boxWidth: 12, font: { size: 11 } } },
  },
  scales: {
    y: { beginAtZero: true, ticks: { precision: 0 }, grid: { color: 'rgba(0,0,0,0.05)' } },
    x: { grid: { display: false } },
  },
}

const trafficChartOptions = {
  ...lineChartOptions,
  plugins: {
    ...lineChartOptions.plugins,
    tooltip: {
      callbacks: {
        label: (ctx) => ` ${ctx.parsed.y} MB`,
      },
    },
  },
}

const horizontalBarOptions = {
  ...baseChartOptions,
  indexAxis: 'y',
  plugins: { legend: { display: false } },
  scales: {
    x: { beginAtZero: true, ticks: { precision: 0 }, grid: { color: 'rgba(0,0,0,0.05)' } },
    y: { grid: { display: false } },
  },
}

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: '62%',
  plugins: {
    legend: { position: 'bottom', labels: { boxWidth: 12, font: { size: 11 } } },
  },
}

const failChartOptions = {
  ...lineChartOptions,
  plugins: { legend: { display: false } },
}

async function loadStats(manual = false) {
  if (!stats.value.summary && !manual) initialLoading.value = true
  else refreshing.value = true
  try {
    const { data } = await dashboardAPI.getStats()
    stats.value = data
    lastUpdated.value = new Date()
  } catch (err) {
    console.error('Failed to load dashboard stats:', err)
  } finally {
    initialLoading.value = false
    refreshing.value = false
  }
}

function formatDate(dateStr) {
  if (!dateStr) return '—'
  try { return format(parseISO(dateStr), 'MMM d, HH:mm:ss') } catch { return dateStr }
}

function formatMAC(mac) {
  if (!mac) return '—'
  return mac.toUpperCase()
}

function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i]
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

function formatRelative(dateStr) {
  if (!dateStr) return '—'
  try { return formatDistanceToNow(parseISO(dateStr), { addSuffix: true }) } catch { return dateStr }
}

function nasDot(status) {
  return { up: 'bg-green-500', down: 'bg-red-500' }[status] || 'bg-gray-400'
}

function nasStatusColor(status) {
  return { up: 'text-green-600', down: 'text-red-600' }[status] || 'text-gray-500'
}

function severityBadge(severity) {
  return {
    critical: 'badge-red',
    high: 'badge-red',
    medium: 'badge-yellow',
    low: 'badge-blue',
  }[severity] || 'badge-gray'
}

function formatAlertType(type) {
  if (!type) return 'Alert'
  return type.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

function setupRefreshTimer() {
  clearInterval(refreshTimer)
  refreshTimer = setInterval(() => loadStats(), refreshInterval.value)
}

watch(refreshInterval, setupRefreshTimer)

onMounted(() => {
  loadStats()
  setupRefreshTimer()
})

onUnmounted(() => clearInterval(refreshTimer))
</script>
