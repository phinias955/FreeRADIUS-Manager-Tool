<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <ShieldExclamationIcon class="w-7 h-7 text-red-500" />
          Security Center
        </h1>
        <p class="text-sm text-gray-500 mt-1">Real-time threat detection and response</p>
      </div>
      <button @click="loadAll" :disabled="loading" class="btn-secondary flex items-center gap-2">
        <ArrowPathIcon class="w-4 h-4" :class="{ 'animate-spin': loading }" />
        Refresh
      </button>
    </div>

    <!-- Stat cards -->
    <div class="grid grid-cols-2 md:grid-cols-5 gap-4">
      <div v-for="stat in summary.stats" :key="stat.label"
        class="bg-white dark:bg-gray-800 rounded-xl shadow p-4 flex flex-col gap-1 border-l-4"
        :class="borderClass(stat.color)">
        <span class="text-2xl font-bold" :class="textClass(stat.color)">{{ stat.value }}</span>
        <span class="text-xs text-gray-500">{{ stat.label }}</span>
      </div>
    </div>

    <!-- Tabs -->
    <div class="border-b border-gray-200 dark:border-gray-700">
      <nav class="-mb-px flex gap-6">
        <button v-for="tab in tabs" :key="tab.id"
          @click="activeTab = tab.id"
          class="pb-2 text-sm font-medium border-b-2 transition-colors"
          :class="activeTab === tab.id
            ? 'border-red-500 text-red-600 dark:text-red-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'">
          {{ tab.label }}
          <span v-if="tab.badge" class="ml-1.5 rounded-full bg-red-100 text-red-700 px-1.5 py-0.5 text-xs">
            {{ tab.badge }}
          </span>
        </button>
      </nav>
    </div>

    <!-- ── Alerts tab ────────────────────────────────────────────────────── -->
    <div v-if="activeTab === 'alerts'" class="space-y-4">
      <div class="flex flex-wrap gap-3 items-center">
        <select v-model="alertFilter.severity" class="input-sm">
          <option value="">All Severities</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
        <select v-model="alertFilter.type" class="input-sm">
          <option value="">All Types</option>
          <option value="honeypot_probe">Honeypot Probe</option>
          <option value="credential_stuffing">Credential Stuffing</option>
          <option value="credential_stuffing_pattern">CS Pattern</option>
        </select>
        <label class="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400">
          <input type="checkbox" v-model="alertFilter.unread" class="rounded" />
          Unread only
        </label>
        <div class="ml-auto flex gap-2">
          <button @click="loadAlerts" class="btn-secondary text-sm">Filter</button>
          <button @click="ackAll" class="btn-primary text-sm">Acknowledge All</button>
        </div>
      </div>

      <div class="bg-white dark:bg-gray-800 rounded-xl shadow overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="th">Severity</th>
              <th class="th">Type</th>
              <th class="th">IP</th>
              <th class="th">Username</th>
              <th class="th">Details</th>
              <th class="th">Time</th>
              <th class="th">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in alerts" :key="a.id" class="tr-hover"
              :class="{ 'bg-red-50 dark:bg-red-900/10': !a.is_acknowledged }">
              <td class="td">
                <span class="badge" :class="severityBadge(a.severity)">{{ a.severity }}</span>
              </td>
              <td class="td font-mono text-xs">{{ a.alert_type }}</td>
              <td class="td font-mono">{{ a.ip_address || '—' }}</td>
              <td class="td">{{ a.username || '—' }}</td>
              <td class="td max-w-xs truncate text-xs text-gray-500">{{ formatDetails(a.details) }}</td>
              <td class="td text-xs text-gray-500">{{ formatDate(a.created_at) }}</td>
              <td class="td">
                <div class="flex gap-1">
                  <button v-if="!a.is_acknowledged" @click="ackAlert(a.id)"
                    class="btn-xs bg-green-100 text-green-700 hover:bg-green-200">Ack</button>
                  <button @click="deleteAlert(a.id)"
                    class="btn-xs bg-red-100 text-red-700 hover:bg-red-200">Del</button>
                </div>
              </td>
            </tr>
            <tr v-if="alerts.length === 0">
              <td colspan="7" class="td text-center text-gray-400 py-8">No alerts</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ── Blocked IPs tab ───────────────────────────────────────────────── -->
    <div v-if="activeTab === 'blocks'" class="space-y-4">
      <div class="flex justify-between items-center">
        <span class="text-sm text-gray-600 dark:text-gray-400">{{ blockedCount }} active block(s)</span>
        <button @click="showBlockModal = true" class="btn-primary text-sm flex items-center gap-1">
          <PlusIcon class="w-4 h-4" /> Block IP
        </button>
      </div>

      <div class="bg-white dark:bg-gray-800 rounded-xl shadow overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="th">IP Address</th>
              <th class="th">Fail Count</th>
              <th class="th">Blocked Until</th>
              <th class="th">Reason</th>
              <th class="th">Auto</th>
              <th class="th">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in blockedIPs" :key="b.id" class="tr-hover">
              <td class="td font-mono">{{ b.ip_address }}</td>
              <td class="td text-center">{{ b.fail_count }}</td>
              <td class="td text-xs">{{ b.blocked_until ? formatDate(b.blocked_until) : 'Permanent' }}</td>
              <td class="td text-xs text-gray-500">{{ b.reason || '—' }}</td>
              <td class="td">
                <span :class="b.auto_blocked ? 'text-orange-500' : 'text-blue-500'" class="text-xs font-medium">
                  {{ b.auto_blocked ? 'Auto' : 'Manual' }}
                </span>
              </td>
              <td class="td">
                <button @click="unblockIP(b.id)" class="btn-xs bg-green-100 text-green-700 hover:bg-green-200">
                  Unblock
                </button>
              </td>
            </tr>
            <tr v-if="blockedIPs.length === 0">
              <td colspan="6" class="td text-center text-gray-400 py-8">No blocked IPs</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ── GeoIP tab ─────────────────────────────────────────────────────── -->
    <div v-if="activeTab === 'geoip'" class="space-y-4">
      <!-- Lookup tool -->
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
        <h3 class="font-semibold text-gray-900 dark:text-white mb-3">IP Lookup</h3>
        <div class="flex gap-3">
          <input v-model="lookupIP" placeholder="Enter IP address" class="input flex-1" />
          <button @click="doLookup" :disabled="lookupLoading" class="btn-primary">
            {{ lookupLoading ? 'Looking up…' : 'Lookup' }}
          </button>
        </div>
        <div v-if="lookupResult" class="mt-3 grid grid-cols-2 md:grid-cols-4 gap-3">
          <div v-for="(val, key) in lookupResultDisplay" :key="key" class="bg-gray-50 dark:bg-gray-700 rounded p-2">
            <div class="text-xs text-gray-500">{{ key }}</div>
            <div class="text-sm font-medium text-gray-900 dark:text-white">{{ val }}</div>
          </div>
        </div>
      </div>

      <!-- Rules -->
      <div class="flex justify-between items-center">
        <h3 class="font-semibold text-gray-900 dark:text-white">Country Rules</h3>
        <button @click="showGeoIPModal = true" class="btn-primary text-sm flex items-center gap-1">
          <PlusIcon class="w-4 h-4" /> Add Rule
        </button>
      </div>

      <div class="bg-white dark:bg-gray-800 rounded-xl shadow overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="th">Code</th>
              <th class="th">Country</th>
              <th class="th">Action</th>
              <th class="th">Active</th>
              <th class="th">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in geoRules" :key="r.id" class="tr-hover">
              <td class="td font-mono font-bold">{{ r.country_code }}</td>
              <td class="td">{{ r.country_name }}</td>
              <td class="td">
                <span class="badge" :class="actionBadge(r.action)">{{ r.action }}</span>
              </td>
              <td class="td">
                <span :class="r.is_active ? 'text-green-500' : 'text-gray-400'" class="text-xs font-medium">
                  {{ r.is_active ? 'Yes' : 'No' }}
                </span>
              </td>
              <td class="td">
                <button @click="deleteRule(r.id)" class="btn-xs bg-red-100 text-red-700 hover:bg-red-200">Delete</button>
              </td>
            </tr>
            <tr v-if="geoRules.length === 0">
              <td colspan="5" class="td text-center text-gray-400 py-8">No rules configured</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ── Auth failure trend chart ──────────────────────────────────────── -->
    <div v-if="activeTab === 'alerts' && summary.fail_trend?.length" class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
      <h3 class="font-semibold text-gray-900 dark:text-white mb-3">Auth Failure Trend (24h)</h3>
      <div class="flex items-end gap-1 h-24">
        <div v-for="pt in summary.fail_trend" :key="pt.hour"
          class="flex-1 bg-red-400 rounded-t relative group"
          :style="{ height: barHeight(pt.fails) + 'px' }">
          <div class="absolute -top-6 left-1/2 -translate-x-1/2 text-xs bg-gray-800 text-white px-1 rounded opacity-0 group-hover:opacity-100 whitespace-nowrap">
            {{ pt.hour }}: {{ pt.fails }}
          </div>
        </div>
      </div>
    </div>

    <!-- ── Block IP modal ────────────────────────────────────────────────── -->
    <div v-if="showBlockModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="showBlockModal = false">
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md p-6 space-y-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Block IP Address</h2>
          <div class="space-y-3">
            <input v-model="blockForm.ip" placeholder="IP Address" class="input w-full" />
            <input v-model.number="blockForm.duration" type="number" min="1" placeholder="Duration (hours)" class="input w-full" />
            <input v-model="blockForm.reason" placeholder="Reason" class="input w-full" />
          </div>
          <div class="flex justify-end gap-3">
            <button @click="showBlockModal = false" class="btn-secondary">Cancel</button>
            <button @click="submitBlock" class="btn-primary">Block</button>
          </div>
        </div>
    </div>

    <!-- ── GeoIP Rule modal ──────────────────────────────────────────────── -->
    <div v-if="showGeoIPModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="showGeoIPModal = false">
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md p-6 space-y-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Add GeoIP Rule</h2>
          <div class="space-y-3">
            <input v-model="geoForm.country_code" placeholder="Country Code (e.g. CN)" maxlength="2" class="input w-full uppercase" />
            <input v-model="geoForm.country_name" placeholder="Country Name" class="input w-full" />
            <select v-model="geoForm.action" class="input w-full">
              <option value="block">Block</option>
              <option value="flag">Flag (alert only)</option>
              <option value="allow">Allow (whitelist)</option>
            </select>
          </div>
          <div class="flex justify-end gap-3">
            <button @click="showGeoIPModal = false" class="btn-secondary">Cancel</button>
            <button @click="submitGeoRule" class="btn-primary">Save Rule</button>
          </div>
        </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ShieldExclamationIcon, ArrowPathIcon, PlusIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'
import { securityAPI } from '../api'

const toast = useToast()
const loading = ref(false)
const activeTab = ref('alerts')

const summary = ref({ stats: [], fail_trend: [] })
const alerts = ref([])
const alertsTotal = ref(0)
const unreadCount = ref(0)
const blockedIPs = ref([])
const blockedCount = ref(0)
const geoRules = ref([])

const alertFilter = ref({ severity: '', type: '', unread: false })
const lookupIP = ref('')
const lookupResult = ref(null)
const lookupLoading = ref(false)
const showBlockModal = ref(false)
const showGeoIPModal = ref(false)
const blockForm = ref({ ip: '', duration: 24, reason: '' })
const geoForm = ref({ country_code: '', country_name: '', action: 'block' })

const tabs = computed(() => [
  { id: 'alerts', label: 'Alerts', badge: unreadCount.value > 0 ? unreadCount.value : null },
  { id: 'blocks', label: 'Blocked IPs', badge: blockedCount.value > 0 ? blockedCount.value : null },
  { id: 'geoip', label: 'GeoIP Rules' },
])

async function loadAll() {
  loading.value = true
  await Promise.all([loadSummary(), loadAlerts(), loadBlocked(), loadGeoRules()])
  loading.value = false
}

async function loadSummary() {
  const r = await securityAPI.getSummary()
  summary.value = r.data
}

async function loadAlerts() {
  const params = {
    severity: alertFilter.value.severity,
    type: alertFilter.value.type,
    unread: alertFilter.value.unread,
    limit: 50
  }
  const r = await securityAPI.listAlerts(params)
  alerts.value = r.data.data || []
  alertsTotal.value = r.data.total || 0
  unreadCount.value = r.data.summary?.unread || 0
}

async function loadBlocked() {
  const r = await securityAPI.getBlockedIPs()
  blockedIPs.value = r.data.data || []
  blockedCount.value = r.data.active_blocks || 0
}

async function loadGeoRules() {
  const r = await securityAPI.listGeoIPRules()
  geoRules.value = r.data.data || []
}

async function ackAlert(id) {
  await securityAPI.ackAlert(id)
  loadAlerts()
}

async function ackAll() {
  await securityAPI.ackAllAlerts()
  toast.success('All alerts acknowledged')
  loadAlerts()
}

async function deleteAlert(id) {
  await securityAPI.deleteAlert(id)
  alerts.value = alerts.value.filter(a => a.id !== id)
}

async function unblockIP(id) {
  await securityAPI.unblockIP(id)
  toast.success('IP unblocked')
  loadBlocked()
}

async function submitBlock() {
  if (!blockForm.value.ip) return
  await securityAPI.blockIP(blockForm.value)
  toast.success('IP blocked')
  showBlockModal.value = false
  blockForm.value = { ip: '', duration: 24, reason: '' }
  loadBlocked()
}

async function doLookup() {
  if (!lookupIP.value) return
  lookupLoading.value = true
  try {
    const r = await securityAPI.geoipLookup(lookupIP.value)
    lookupResult.value = r.data
  } catch {
    toast.error('Lookup failed')
  }
  lookupLoading.value = false
}

const lookupResultDisplay = computed(() => {
  if (!lookupResult.value) return {}
  const r = lookupResult.value
  return {
    'Country': `${r.country_name} (${r.country_code})`,
    'City': r.city || 'Unknown',
    'ISP': r.isp || 'Unknown',
    'VPN/Proxy': r.is_vpn ? 'Yes ⚠️' : 'No',
    'Cached': r.cached ? 'Yes' : 'No',
  }
})

async function submitGeoRule() {
  if (!geoForm.value.country_code || !geoForm.value.country_name) return
  await securityAPI.createGeoIPRule(geoForm.value)
  toast.success('GeoIP rule saved')
  showGeoIPModal.value = false
  geoForm.value = { country_code: '', country_name: '', action: 'block' }
  loadGeoRules()
}

async function deleteRule(id) {
  await securityAPI.deleteGeoIPRule(id)
  geoRules.value = geoRules.value.filter(r => r.id !== id)
}

function severityBadge(s) {
  return {
    critical: 'bg-red-100 text-red-700',
    high:     'bg-orange-100 text-orange-700',
    medium:   'bg-yellow-100 text-yellow-700',
    low:      'bg-blue-100 text-blue-700',
  }[s] || 'bg-gray-100 text-gray-700'
}

function actionBadge(a) {
  return {
    block: 'bg-red-100 text-red-700',
    flag:  'bg-yellow-100 text-yellow-700',
    allow: 'bg-green-100 text-green-700',
  }[a] || 'bg-gray-100 text-gray-700'
}

function borderClass(color) {
  return {
    red:    'border-red-500',
    orange: 'border-orange-500',
    yellow: 'border-yellow-500',
    purple: 'border-purple-500',
    green:  'border-green-500',
  }[color] || 'border-gray-300'
}

function textClass(color) {
  return {
    red:    'text-red-600',
    orange: 'text-orange-600',
    yellow: 'text-yellow-600',
    purple: 'text-purple-600',
    green:  'text-green-600',
  }[color] || 'text-gray-700'
}

function formatDate(d) {
  return new Date(d).toLocaleString()
}

function formatDetails(d) {
  if (!d) return '—'
  try { return JSON.stringify(JSON.parse(d), null, 0).slice(0, 80) } catch { return d }
}

const maxFails = computed(() => Math.max(...(summary.value.fail_trend || []).map(t => t.fails), 1))
function barHeight(fails) { return Math.max(4, Math.round((fails / maxFails.value) * 88)) }

onMounted(loadAll)
</script>

<style scoped>
.input { @apply block w-full border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500; }
.input-sm { @apply border border-gray-300 dark:border-gray-600 rounded px-2 py-1 text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white; }
.btn-primary { @apply bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50; }
.btn-secondary { @apply bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 px-4 py-2 rounded-lg text-sm font-medium transition-colors; }
.btn-xs { @apply px-2 py-0.5 text-xs rounded font-medium transition-colors; }
.th { @apply px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider; }
.td { @apply px-4 py-3 text-sm text-gray-900 dark:text-gray-100; }
.tr-hover { @apply hover:bg-gray-50 dark:hover:bg-gray-700/50 border-b border-gray-100 dark:border-gray-700; }
.badge { @apply inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium; }
</style>
