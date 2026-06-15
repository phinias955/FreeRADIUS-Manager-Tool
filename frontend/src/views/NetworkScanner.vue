<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Network Scanner</h1>
        <p class="text-sm text-gray-500 mt-0.5">Discover access points, routers, and network devices with nmap</p>
      </div>
      <button @click="loadHistory" class="btn-secondary" :disabled="loadingHistory">
        <ArrowPathIcon class="w-4 h-4" :class="{ spinner: loadingHistory }" />
        Refresh
      </button>
    </div>

    <!-- nmap status -->
    <div v-if="!scannerStatus.nmap_available" class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-amber-800 text-sm">
      nmap is not available on the server. Rebuild the backend container with nmap installed.
    </div>

    <!-- Scan launcher -->
    <div class="card">
      <h3 class="font-semibold text-gray-900 mb-4">Start New Scan</h3>
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div>
          <label class="form-label">Subnet (CIDR)</label>
          <input
            v-model="subnet"
            type="text"
            class="form-input font-mono"
            placeholder="10.10.0.0/18"
            :disabled="scanning"
          />
          <p class="text-xs text-gray-400 mt-1">
            Up to {{ scannerStatus.max_subnet || '/18' }} ({{ formatHostCount(scannerStatus.max_hosts) }} addresses).
            Large subnets use two-phase scanning and may take 30–90 minutes.
          </p>
          <p v-if="subnetHostEstimate > 256" class="text-xs text-amber-600 mt-1">
            {{ formatHostCount(subnetHostEstimate) }} addresses — port scans run on live hosts only (max 512).
          </p>
        </div>
        <div>
          <label class="form-label">Scan Profile</label>
          <select v-model="scanType" class="form-select" :disabled="scanning">
            <option v-for="t in scannerStatus.scan_types || []" :key="t.id" :value="t.id">
              {{ t.label }}
            </option>
          </select>
          <p class="text-xs text-gray-400 mt-1">{{ selectedTypeDescription }}</p>
        </div>
        <div class="flex items-end">
          <button
            @click="startScan"
            class="btn-primary w-full justify-center"
            :disabled="scanning || !subnet || !scannerStatus.nmap_available"
          >
            <MagnifyingGlassCircleIcon class="w-5 h-5" />
            {{ scanning ? 'Scanning…' : 'Start Scan' }}
          </button>
        </div>
      </div>

      <div v-if="scanning" class="mt-4 p-4 rounded-lg bg-blue-50 border border-blue-100">
        <div class="flex items-center gap-3">
          <span class="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full spinner"></span>
          <div>
            <p class="text-sm font-medium text-blue-800">Scan in progress — {{ activeScan?.subnet }} ({{ activeScan?.scan_type }})</p>
            <p class="text-xs text-blue-600">Large scans may take 30–90 minutes. Keep this page open or check Scan History.</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Results summary -->
    <div v-if="activeScan?.status === 'completed'" class="grid grid-cols-2 sm:grid-cols-5 gap-3">
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-gray-900">{{ resultSummary.total || 0 }}</p>
        <p class="text-xs text-gray-500 uppercase">Devices Found</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-purple-600">{{ resultSummary.access_points || 0 }}</p>
        <p class="text-xs text-gray-500 uppercase">Access Points</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-orange-600">{{ resultSummary.routers || 0 }}</p>
        <p class="text-xs text-gray-500 uppercase">Routers</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-green-600">{{ resultSummary.radius_capable || 0 }}</p>
        <p class="text-xs text-gray-500 uppercase">RADIUS Capable</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-sm font-medium text-gray-600">{{ formatDate(activeScan.finished_at) }}</p>
        <p class="text-xs text-gray-500 uppercase">Completed</p>
      </div>
    </div>

    <!-- Failed scan -->
    <div v-if="activeScan?.status === 'failed'" class="card border-red-200 bg-red-50">
      <p class="text-sm font-medium text-red-800">Scan failed</p>
      <p class="text-xs text-red-600 mt-1">{{ activeScan.error_message }}</p>
    </div>

    <!-- Device filters + results -->
    <div v-if="hosts.length" class="card p-0 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-b border-gray-200">
        <div class="flex flex-wrap gap-2">
          <button
            v-for="f in filters"
            :key="f.id"
            @click="deviceFilter = f.id"
            class="px-3 py-1 text-xs rounded-full border transition-colors"
            :class="deviceFilter === f.id
              ? 'bg-blue-600 text-white border-blue-600'
              : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300'"
          >
            {{ f.label }} ({{ filterCount(f.id) }})
          </button>
        </div>
        <input v-model="search" type="text" class="form-input w-48 text-sm py-1.5" placeholder="Search IP, MAC, vendor…" />
      </div>

      <div class="table-container rounded-none border-0">
        <table class="table">
          <thead>
            <tr>
              <th>IP Address</th>
              <th>Hostname</th>
              <th>MAC / Vendor</th>
              <th>Device Type</th>
              <th>Open Ports</th>
              <th>Flags</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="h in filteredHosts" :key="h.id">
              <td class="font-mono text-sm font-medium">{{ h.ip_address }}</td>
              <td class="text-sm text-gray-600">{{ h.hostname || '—' }}</td>
              <td>
                <p class="font-mono text-xs text-gray-500">{{ h.mac_address || '—' }}</p>
                <p class="text-xs text-gray-400">{{ h.vendor || '' }}</p>
              </td>
              <td>
                <span class="badge" :class="deviceBadge(h.device_type)">{{ deviceLabel(h.device_type) }}</span>
              </td>
              <td class="font-mono text-xs text-gray-500 max-w-[140px] truncate" :title="(h.open_ports || []).join(', ')">
                {{ (h.open_ports || []).slice(0, 8).join(', ') || '—' }}
              </td>
              <td>
                <span v-if="h.is_access_point" class="badge badge-purple mr-1">AP</span>
                <span v-if="h.is_radius_capable" class="badge badge-green">RADIUS</span>
              </td>
              <td>
                <button
                  @click="importAsNAS(h)"
                  class="text-xs px-2 py-1 rounded bg-blue-100 text-blue-700 hover:bg-blue-200"
                  :disabled="importingId === h.id"
                >
                  {{ importingId === h.id ? '…' : 'Add NAS' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Scan history -->
    <div class="card p-0 overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-200">
        <h3 class="font-semibold text-gray-900">Scan History</h3>
      </div>
      <div class="table-container rounded-none border-0">
        <table class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Subnet</th>
              <th>Profile</th>
              <th>Status</th>
              <th>Hosts</th>
              <th>Started</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!scanHistory.length">
              <td colspan="7" class="text-center text-gray-400 py-8">No scans yet</td>
            </tr>
            <tr
              v-for="s in scanHistory"
              :key="s.id"
              class="cursor-pointer"
              :class="{ 'bg-blue-50': activeScan?.id === s.id }"
              @click="loadScan(s.id)"
            >
              <td class="text-gray-500">#{{ s.id }}</td>
              <td class="font-mono text-sm">{{ s.subnet }}</td>
              <td class="text-sm capitalize">{{ s.scan_type }}</td>
              <td><span class="badge" :class="statusBadge(s.status)">{{ s.status }}</span></td>
              <td>{{ s.host_count }}</td>
              <td class="text-xs text-gray-500">{{ formatDate(s.started_at) }}</td>
              <td>
                <button @click.stop="deleteScan(s.id)" class="text-xs text-red-600 hover:underline">Delete</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { networkScannerAPI } from '@/api'
import { ArrowPathIcon, MagnifyingGlassCircleIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'
import { format, parseISO } from 'date-fns'

const toast = useToast()

const subnet = ref('10.10.0.0/18')
const scanType = ref('ap')
const scanning = ref(false)
const loadingHistory = ref(false)
const importingId = ref(null)
const deviceFilter = ref('all')
const search = ref('')

const scannerStatus = ref({ nmap_available: false, scan_types: [] })
const scanHistory = ref([])
const activeScan = ref(null)
const hosts = ref([])
const resultSummary = ref({})

let pollTimer = null

const filters = [
  { id: 'all', label: 'All' },
  { id: 'ap', label: 'Access Points' },
  { id: 'router', label: 'Routers' },
  { id: 'radius', label: 'RADIUS' },
  { id: 'other', label: 'Other' },
]

const selectedTypeDescription = computed(() => {
  const t = (scannerStatus.value.scan_types || []).find(x => x.id === scanType.value)
  return t?.description || ''
})

const subnetHostEstimate = computed(() => {
  try {
    const [ip, prefix] = subnet.value.trim().split('/')
    if (!ip || !prefix) return 0
    const p = parseInt(prefix, 10)
    if (p < 0 || p > 32) return 0
    return Math.pow(2, 32 - p)
  } catch {
    return 0
  }
})

function formatHostCount(n) {
  if (!n) return '—'
  if (n >= 1000) return n.toLocaleString()
  return String(n)
}

const filteredHosts = computed(() => {
  let list = hosts.value
  if (deviceFilter.value === 'ap') list = list.filter(h => h.is_access_point)
  else if (deviceFilter.value === 'router') list = list.filter(h => h.device_type === 'router')
  else if (deviceFilter.value === 'radius') list = list.filter(h => h.is_radius_capable)
  else if (deviceFilter.value === 'other') list = list.filter(h => !h.is_access_point && h.device_type !== 'router' && !h.is_radius_capable)

  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(h =>
      h.ip_address?.includes(q) ||
      h.hostname?.toLowerCase().includes(q) ||
      h.mac_address?.toLowerCase().includes(q) ||
      h.vendor?.toLowerCase().includes(q)
    )
  }
  return list
})

function filterCount(id) {
  if (id === 'all') return hosts.value.length
  if (id === 'ap') return hosts.value.filter(h => h.is_access_point).length
  if (id === 'router') return hosts.value.filter(h => h.device_type === 'router').length
  if (id === 'radius') return hosts.value.filter(h => h.is_radius_capable).length
  return hosts.value.filter(h => !h.is_access_point && h.device_type !== 'router' && !h.is_radius_capable).length
}

async function loadStatus() {
  try {
    const { data } = await networkScannerAPI.status()
    scannerStatus.value = data
    if (data.scan_types?.length && !data.scan_types.find(t => t.id === scanType.value)) {
      scanType.value = data.scan_types[0].id
    }
  } catch { /* silent */ }
}

async function loadHistory() {
  loadingHistory.value = true
  try {
    const { data } = await networkScannerAPI.listScans({ limit: 20 })
    scanHistory.value = data.data || []
  } catch {
    toast.error('Failed to load scan history')
  } finally {
    loadingHistory.value = false
  }
}

async function loadScan(id) {
  try {
    const { data } = await networkScannerAPI.getScan(id)
    activeScan.value = data.scan
    hosts.value = data.hosts || []
    resultSummary.value = data.summary || {}
    if (data.scan.status === 'running') {
      scanning.value = true
      startPolling(id)
    } else {
      scanning.value = false
      stopPolling()
    }
  } catch {
    toast.error('Failed to load scan results')
  }
}

async function startScan() {
  scanning.value = true
  hosts.value = []
  resultSummary.value = {}
  try {
    const { data } = await networkScannerAPI.startScan({
      subnet: subnet.value.trim(),
      scan_type: scanType.value,
    })
    toast.success('Network scan started')
    await loadHistory()
    activeScan.value = { id: data.id, subnet: data.subnet, scan_type: data.scan_type, status: 'running' }
    startPolling(data.id)
  } catch (err) {
    scanning.value = false
    toast.error(err.response?.data?.error || 'Failed to start scan')
  }
}

function startPolling(id) {
  stopPolling()
  pollTimer = setInterval(async () => {
    try {
      const { data } = await networkScannerAPI.getScan(id)
      activeScan.value = data.scan
      hosts.value = data.hosts || []
      resultSummary.value = data.summary || {}
      if (data.scan.status !== 'running') {
        scanning.value = false
        stopPolling()
        loadHistory()
        if (data.scan.status === 'completed') {
          toast.success(`Scan complete — ${data.summary?.total || 0} devices found`)
        } else if (data.scan.status === 'failed') {
          toast.error(data.scan.error_message || 'Scan failed')
        }
      }
    } catch { /* keep polling */ }
  }, 2500)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function deleteScan(id) {
  if (!confirm('Delete this scan and all discovered hosts?')) return
  try {
    await networkScannerAPI.deleteScan(id)
    if (activeScan.value?.id === id) {
      activeScan.value = null
      hosts.value = []
    }
    loadHistory()
    toast.success('Scan deleted')
  } catch {
    toast.error('Failed to delete scan')
  }
}

async function importAsNAS(host) {
  importingId.value = host.id
  try {
    const { data } = await networkScannerAPI.importAsNAS(host.id)
    toast.success(`NAS device created: ${data.nasname}`)
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed to import as NAS')
  } finally {
    importingId.value = null
  }
}

function formatDate(d) {
  if (!d) return '—'
  try { return format(parseISO(d), 'MMM d, HH:mm') } catch { return d }
}

function statusBadge(s) {
  return { running: 'badge-blue', completed: 'badge-green', failed: 'badge-red' }[s] || 'badge-gray'
}

function deviceBadge(t) {
  return {
    access_point: 'badge-purple',
    router: 'badge-orange',
    switch: 'badge-blue',
    network_device: 'badge-blue',
    server: 'badge-gray',
    web_device: 'badge-yellow',
  }[t] || 'badge-gray'
}

function deviceLabel(t) {
  return (t || 'unknown').replace(/_/g, ' ')
}

onMounted(async () => {
  await loadStatus()
  await loadHistory()
})

onUnmounted(stopPolling)
</script>
