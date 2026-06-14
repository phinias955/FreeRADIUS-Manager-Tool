<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Network Map</h1>
        <p class="text-sm text-gray-500 mt-0.5">Live topology of all NAS devices and their status</p>
      </div>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5 text-xs text-gray-500">
          <span class="w-2 h-2 rounded-full bg-green-500"></span> Online
          <span class="w-2 h-2 rounded-full bg-red-500 ml-2"></span> Offline
          <span class="w-2 h-2 rounded-full bg-gray-400 ml-2"></span> Unknown
        </div>
        <button @click="load" class="btn-secondary text-xs py-1.5 px-3">
          <ArrowPathIcon class="w-3.5 h-3.5" />
          Refresh
        </button>
      </div>
    </div>

    <!-- Live stats bar -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <div class="card p-4 flex items-center gap-3">
        <div class="w-9 h-9 bg-green-100 rounded-xl flex items-center justify-center">
          <ServerIcon class="w-5 h-5 text-green-600" />
        </div>
        <div>
          <p class="text-xs text-gray-500">Online NAS</p>
          <p class="text-xl font-bold text-green-600">{{ onlineCount }}</p>
        </div>
      </div>
      <div class="card p-4 flex items-center gap-3">
        <div class="w-9 h-9 bg-red-100 rounded-xl flex items-center justify-center">
          <ServerIcon class="w-5 h-5 text-red-600" />
        </div>
        <div>
          <p class="text-xs text-gray-500">Offline NAS</p>
          <p class="text-xl font-bold text-red-600">{{ offlineCount }}</p>
        </div>
      </div>
      <div class="card p-4 flex items-center gap-3">
        <div class="w-9 h-9 bg-blue-100 rounded-xl flex items-center justify-center">
          <UsersIcon class="w-5 h-5 text-blue-600" />
        </div>
        <div>
          <p class="text-xs text-gray-500">Active Sessions</p>
          <p class="text-xl font-bold text-blue-600">{{ liveStats.active_sessions || 0 }}</p>
        </div>
      </div>
      <div class="card p-4 flex items-center gap-3">
        <div class="w-9 h-9 bg-purple-100 rounded-xl flex items-center justify-center">
          <SignalIcon class="w-5 h-5 text-purple-600" />
        </div>
        <div>
          <p class="text-xs text-gray-500">Bandwidth In</p>
          <p class="text-xl font-bold text-purple-600">{{ (liveStats.bandwidth_in_mbps || 0).toFixed(1) }}<span class="text-sm font-normal">Mbps</span></p>
        </div>
      </div>
    </div>

    <!-- Topology map -->
    <div class="card p-6">
      <div v-if="loading" class="text-center py-16 text-gray-400">Loading topology…</div>
      <div v-else-if="!devices.length" class="text-center py-16 text-gray-400">
        <ServerIcon class="w-12 h-12 mx-auto mb-3 text-gray-300" />
        <p>No NAS devices configured</p>
      </div>
      <div v-else>
        <!-- Central RADIUS server node -->
        <div class="flex flex-col items-center mb-8">
          <div class="w-20 h-20 bg-blue-600 rounded-2xl flex items-center justify-center shadow-lg shadow-blue-200">
            <ServerIcon class="w-10 h-10 text-white" />
          </div>
          <p class="mt-2 font-bold text-blue-700 text-sm">FreeRADIUS Server</p>
          <p class="text-xs text-gray-400">Authentication Hub</p>

          <!-- Vertical connector line -->
          <div class="w-px h-8 bg-gray-300 mt-2"></div>
          <div class="w-px h-0 border-l-2 border-dashed border-gray-300"></div>
        </div>

        <!-- NAS device nodes -->
        <div class="flex flex-wrap justify-center gap-4">
          <div v-for="dev in devices" :key="dev.id"
            class="flex flex-col items-center group relative"
            style="min-width: 120px; max-width: 140px">
            <!-- Connector line up -->
            <div class="w-px h-6 mb-2" :class="dev.ping_status === 'up' ? 'bg-green-400' : dev.ping_status === 'down' ? 'bg-red-400' : 'bg-gray-300'"></div>

            <!-- Device card -->
            <div class="w-full rounded-xl border-2 p-3 text-center transition-all cursor-pointer hover:shadow-md"
              :class="statusBorder(dev.ping_status)"
              @click="selectedDev = selectedDev?.id === dev.id ? null : dev">
              <!-- Status dot -->
              <div class="flex justify-center mb-2">
                <span class="w-3 h-3 rounded-full" :class="statusDot(dev.ping_status)"></span>
              </div>
              <div class="w-10 h-10 rounded-lg flex items-center justify-center mx-auto mb-2"
                :class="statusBg(dev.ping_status)">
                <ServerIcon class="w-5 h-5" :class="statusIcon(dev.ping_status)" />
              </div>
              <p class="text-xs font-semibold text-gray-900 truncate">{{ dev.shortname || dev.nasname }}</p>
              <p class="text-xs text-gray-400 font-mono truncate">{{ dev.nasname }}</p>
              <p class="text-xs mt-1 font-medium" :class="dev.ping_status === 'up' ? 'text-green-600' : dev.ping_status === 'down' ? 'text-red-500' : 'text-gray-400'">
                {{ dev.ping_status === 'up' ? dev.ping_latency_ms?.toFixed(0) + 'ms' : dev.ping_status || 'unknown' }}
              </p>
            </div>
          </div>
        </div>

        <!-- Selected device detail -->
        <div v-if="selectedDev" class="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-xl text-sm">
          <div class="flex items-start justify-between">
            <div>
              <p class="font-semibold text-blue-900">{{ selectedDev.shortname || selectedDev.nasname }}</p>
              <p class="font-mono text-blue-700 text-xs mt-0.5">{{ selectedDev.nasname }}</p>
            </div>
            <span class="badge" :class="selectedDev.ping_status === 'up' ? 'badge-green' : 'badge-red'">
              {{ selectedDev.ping_status || 'unknown' }}
            </span>
          </div>
          <div class="grid grid-cols-2 gap-3 mt-3 text-xs text-blue-700">
            <div>Latency: <span class="font-semibold">{{ selectedDev.ping_latency_ms?.toFixed(1) || '—' }}ms</span></div>
            <div>Last ping: <span class="font-semibold">{{ formatDate(selectedDev.last_ping) }}</span></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { nasStatusAPI } from '@/api'
import axios from 'axios'
import { format, parseISO } from 'date-fns'
import { ServerIcon, UsersIcon, SignalIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'

const loading = ref(false)
const devices = ref([])
const selectedDev = ref(null)
const liveStats = ref({})
let refreshTimer
let statsSource

const onlineCount = computed(() => devices.value.filter(d => d.ping_status === 'up').length)
const offlineCount = computed(() => devices.value.filter(d => d.ping_status === 'down').length)

function statusBorder(s) {
  return s === 'up' ? 'border-green-300 bg-white' : s === 'down' ? 'border-red-300 bg-red-50' : 'border-gray-200 bg-gray-50'
}
function statusDot(s) {
  return s === 'up' ? 'bg-green-500 animate-pulse' : s === 'down' ? 'bg-red-500' : 'bg-gray-400'
}
function statusBg(s) {
  return s === 'up' ? 'bg-green-100' : s === 'down' ? 'bg-red-100' : 'bg-gray-100'
}
function statusIcon(s) {
  return s === 'up' ? 'text-green-600' : s === 'down' ? 'text-red-600' : 'text-gray-500'
}
function formatDate(d) {
  if (!d) return 'never'
  try { return format(parseISO(d), 'HH:mm:ss') } catch { return d }
}

async function load() {
  loading.value = true
  try {
    const { data } = await nasStatusAPI.status()
    devices.value = data.data || []
  } catch { /* silent */ } finally { loading.value = false }
}

async function loadCurrentStats() {
  try {
    const token = localStorage.getItem('token') || sessionStorage.getItem('token') || ''
    const { data } = await axios.get('/api/v1/live/stats/current', {
      headers: token ? { Authorization: 'Bearer ' + token } : {}
    })
    liveStats.value = data
  } catch { /* silent */ }
}

onMounted(() => {
  load()
  loadCurrentStats()
  refreshTimer = setInterval(() => { load(); loadCurrentStats() }, 30000)
})
onUnmounted(() => clearInterval(refreshTimer))
</script>
