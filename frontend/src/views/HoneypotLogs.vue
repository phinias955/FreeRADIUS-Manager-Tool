<template>
  <div class="p-6 space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <BugAntIcon class="w-7 h-7 text-purple-500" />
          Honeypot Logs
        </h1>
        <p class="text-sm text-gray-500 mt-1">RADIUS probes captured on the decoy listener (UDP 11812)</p>
      </div>
      <div class="flex gap-3">
        <button @click="clearLogs" class="btn-secondary text-sm flex items-center gap-1">
          <TrashIcon class="w-4 h-4" /> Clear Old Logs
        </button>
        <button @click="load" class="btn-secondary flex items-center gap-2">
          <ArrowPathIcon class="w-4 h-4" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
      </div>
    </div>

    <!-- Status cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
        <div class="text-2xl font-bold text-purple-600">{{ status.today_probes }}</div>
        <div class="text-xs text-gray-500">Probes Today</div>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
        <div class="text-2xl font-bold text-gray-700 dark:text-gray-300">{{ status.total_probes }}</div>
        <div class="text-xs text-gray-500">Total Probes</div>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
        <div class="text-2xl font-bold" :class="status.running ? 'text-green-500' : 'text-red-500'">
          {{ status.running ? 'Running' : 'Stopped' }}
        </div>
        <div class="text-xs text-gray-500">Honeypot Status</div>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
        <div class="text-2xl font-bold text-orange-500">{{ topIPs.length }}</div>
        <div class="text-xs text-gray-500">Unique Attackers</div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Top attacking IPs -->
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
        <h3 class="font-semibold text-gray-900 dark:text-white mb-3">Top Attacking IPs</h3>
        <div class="space-y-2">
          <div v-for="(ip, i) in topIPs" :key="ip.ip" class="flex items-center gap-2">
            <span class="w-5 h-5 rounded-full bg-purple-100 text-purple-700 text-xs flex items-center justify-center font-bold">
              {{ i + 1 }}
            </span>
            <span class="font-mono text-sm text-gray-900 dark:text-white flex-1">{{ ip.ip }}</span>
            <span class="text-xs bg-red-100 text-red-700 px-2 py-0.5 rounded-full">{{ ip.count }}</span>
            <button @click="filterByIP(ip.ip)" class="text-xs text-blue-500 hover:underline">Filter</button>
          </div>
          <p v-if="topIPs.length === 0" class="text-sm text-gray-400">No data yet</p>
        </div>
      </div>

      <!-- Log table -->
      <div class="lg:col-span-2 bg-white dark:bg-gray-800 rounded-xl shadow overflow-hidden">
        <div class="p-3 border-b border-gray-100 dark:border-gray-700 flex gap-2">
          <input v-model="ipFilter" @keyup.enter="load" placeholder="Filter by source IP…" class="input flex-1 text-sm py-1.5" />
          <button @click="ipFilter = ''; load()" class="btn-secondary text-sm py-1.5">Clear</button>
          <button @click="load" class="btn-primary text-sm py-1.5">Apply</button>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead class="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th class="th">Source IP</th>
                <th class="th">Username</th>
                <th class="th">NAS IP</th>
                <th class="th">Attributes</th>
                <th class="th">Time</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in logs" :key="log.id" class="tr-hover">
                <td class="td font-mono text-purple-600 dark:text-purple-400">{{ log.source_ip }}</td>
                <td class="td font-mono">{{ log.username || '—' }}</td>
                <td class="td font-mono text-xs text-gray-500">{{ log.nas_ip || '—' }}</td>
                <td class="td text-xs text-gray-500 max-w-xs truncate">
                  {{ formatAttrs(log.attributes) }}
                </td>
                <td class="td text-xs text-gray-500">{{ formatDate(log.created_at) }}</td>
              </tr>
              <tr v-if="logs.length === 0">
                <td colspan="5" class="td text-center text-gray-400 py-10">
                  <BugAntIcon class="w-8 h-8 mx-auto mb-2 text-gray-300" />
                  <p>No probes detected yet</p>
                  <p class="text-xs">The honeypot listens on UDP port 11812</p>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="total > 0" class="p-3 border-t border-gray-100 dark:border-gray-700 text-xs text-gray-500">
          Showing {{ logs.length }} of {{ total }} events
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { BugAntIcon, ArrowPathIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'
import { securityAPI } from '../api'

const toast = useToast()
const loading = ref(false)
const logs = ref([])
const total = ref(0)
const topIPs = ref([])
const status = ref({ running: false, enabled: false, today_probes: 0, total_probes: 0 })
const ipFilter = ref('')

async function load() {
  loading.value = true
  try {
    const [logsRes, statusRes] = await Promise.all([
      securityAPI.listHoneypotLogs({ ip: ipFilter.value, limit: 100 }),
      securityAPI.honeypotStatus(),
    ])
    logs.value = logsRes.data.data || []
    total.value = logsRes.data.total || 0
    topIPs.value = logsRes.data.top_ips || []
    status.value = statusRes.data
  } catch {
    toast.error('Failed to load honeypot data')
  }
  loading.value = false
}

function filterByIP(ip) {
  ipFilter.value = ip
  load()
}

async function clearLogs() {
  if (!confirm('Delete all honeypot logs older than 30 days?')) return
  const r = await securityAPI.clearHoneypotLogs({ older_than_days: 30 })
  toast.success(`Deleted ${r.data.deleted} entries`)
  load()
}

function formatDate(d) {
  return new Date(d).toLocaleString()
}

function formatAttrs(a) {
  if (!a) return '—'
  try {
    const obj = typeof a === 'string' ? JSON.parse(a) : a
    return Object.entries(obj).map(([k, v]) => `${k}=${v}`).join(', ')
  } catch {
    return String(a)
  }
}

onMounted(load)
</script>

<style scoped>
.input { @apply block border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500; }
.btn-primary { @apply bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors; }
.btn-secondary { @apply bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 px-4 py-2 rounded-lg text-sm font-medium transition-colors; }
.th { @apply px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider; }
.td { @apply px-4 py-3 text-sm text-gray-900 dark:text-gray-100; }
.tr-hover { @apply hover:bg-gray-50 dark:hover:bg-gray-700/50 border-b border-gray-100 dark:border-gray-700; }
</style>
