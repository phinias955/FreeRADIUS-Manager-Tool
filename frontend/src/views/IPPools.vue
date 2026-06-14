<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">IP Pools</h1>
        <p class="text-sm text-gray-500 mt-0.5">Manage IP address pools and assignments for RADIUS users</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
        <PlusIcon class="w-4 h-4" />
        New Pool
      </button>
    </div>

    <!-- Pool cards -->
    <div v-if="loading" class="text-center py-16 text-gray-400">Loading…</div>
    <div v-else-if="!pools.length" class="card text-center py-16 text-gray-400">
      <CircleStackIcon class="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p class="font-medium">No IP pools configured</p>
      <p class="text-sm mt-1">Create a pool to assign static IPs to RADIUS users</p>
    </div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-for="pool in pools" :key="pool.id" class="card">
        <div class="flex items-start justify-between mb-3">
          <div>
            <h3 class="font-semibold text-gray-900">{{ pool.name }}</h3>
            <p class="text-xs font-mono text-blue-600 mt-0.5">{{ pool.network }}</p>
          </div>
          <span class="badge" :class="pool.is_active ? 'badge-green' : 'badge-gray'">
            {{ pool.is_active ? 'Active' : 'Inactive' }}
          </span>
        </div>

        <!-- Usage bar -->
        <div class="mb-3">
          <div class="flex justify-between text-xs text-gray-500 mb-1">
            <span>{{ pool.used_ips }} / {{ pool.total_ips }} used</span>
            <span>{{ pool.free_ips }} free</span>
          </div>
          <div class="w-full bg-gray-100 rounded-full h-2">
            <div class="h-2 rounded-full transition-all"
              :class="usagePct(pool) > 90 ? 'bg-red-500' : usagePct(pool) > 70 ? 'bg-yellow-500' : 'bg-green-500'"
              :style="{ width: usagePct(pool) + '%' }">
            </div>
          </div>
        </div>

        <div class="space-y-1 text-xs text-gray-500 mb-3">
          <div v-if="pool.gateway">Gateway: <span class="font-mono">{{ pool.gateway }}</span></div>
          <div>DNS: <span class="font-mono">{{ pool.dns1 }} / {{ pool.dns2 }}</span></div>
        </div>

        <div class="flex gap-2 pt-3 border-t border-gray-100">
          <button @click="viewIPs(pool)" class="flex-1 btn-secondary text-xs py-1.5">
            <MagnifyingGlassIcon class="w-3.5 h-3.5" />
            View IPs
          </button>
          <button @click="openAssign(pool)" class="flex-1 btn-primary text-xs py-1.5">
            <ArrowPathIcon class="w-3.5 h-3.5" />
            Assign IP
          </button>
          <button v-if="authStore.isAdmin" @click="removePool(pool)" class="p-1.5 rounded-lg border border-red-200 text-red-500 hover:bg-red-50">
            <TrashIcon class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>

    <!-- IP list panel -->
    <div v-if="viewingPool" class="card">
      <div class="flex items-center justify-between mb-4">
        <h3 class="font-semibold text-gray-900">{{ viewingPool.name }} — IP Addresses</h3>
        <div class="flex gap-2">
          <input v-model="ipSearch" type="text" class="form-input py-1 text-sm w-44" placeholder="Filter IPs…" />
          <button @click="viewingPool = null" class="btn-secondary text-xs py-1.5">Close</button>
        </div>
      </div>
      <div class="max-h-96 overflow-y-auto">
        <table class="table">
          <thead>
            <tr>
              <th>IP Address</th>
              <th>Username</th>
              <th>Type</th>
              <th>Leased At</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="ip in filteredIPs" :key="ip.id">
              <td class="font-mono text-sm">{{ ip.ip_address }}</td>
              <td>
                <span v-if="ip.username" class="font-medium text-blue-700">{{ ip.username }}</span>
                <span v-else class="text-gray-400 text-xs">free</span>
              </td>
              <td>
                <span v-if="ip.username" class="badge" :class="ip.is_static ? 'badge-blue' : 'badge-green'">
                  {{ ip.is_static ? 'Static' : 'Dynamic' }}
                </span>
              </td>
              <td class="text-xs text-gray-500">{{ ip.leased_at ? formatDate(ip.leased_at) : '—' }}</td>
              <td>
                <button v-if="ip.username && authStore.isAdmin" @click="release(ip.username)" class="text-xs px-2 py-1 rounded bg-red-100 text-red-600 hover:bg-red-200">
                  Release
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Create Pool Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">New IP Pool</h3>
          <button @click="showCreateModal = false" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="createPool" class="p-6 space-y-4">
          <div>
            <label class="form-label">Pool Name <span class="text-red-500">*</span></label>
            <input v-model="createForm.name" type="text" class="form-input" required placeholder="e.g. Hotspot Pool 1" />
          </div>
          <div>
            <label class="form-label">Network CIDR <span class="text-red-500">*</span></label>
            <input v-model="createForm.network" type="text" class="form-input" required placeholder="10.10.0.0/24" />
            <p class="text-xs text-gray-400 mt-1">Max 1022 IPs will be generated (up to /22)</p>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Gateway (optional)</label>
              <input v-model="createForm.gateway" type="text" class="form-input" placeholder="10.10.0.1" />
            </div>
            <div>
              <label class="form-label">Description</label>
              <input v-model="createForm.description" type="text" class="form-input" placeholder="Optional" />
            </div>
            <div>
              <label class="form-label">DNS 1</label>
              <input v-model="createForm.dns1" type="text" class="form-input" placeholder="8.8.8.8" />
            </div>
            <div>
              <label class="form-label">DNS 2</label>
              <input v-model="createForm.dns2" type="text" class="form-input" placeholder="8.8.4.4" />
            </div>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showCreateModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Create Pool
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Assign IP Modal -->
    <div v-if="showAssignModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-sm">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">Assign IP from {{ assignPool?.name }}</h3>
          <button @click="showAssignModal = false" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="assignIP" class="p-6 space-y-4">
          <div>
            <label class="form-label">Username <span class="text-red-500">*</span></label>
            <input v-model="assignForm.username" type="text" class="form-input" required placeholder="RADIUS username" />
          </div>
          <div>
            <label class="form-label">Specific IP (optional)</label>
            <input v-model="assignForm.ip_address" type="text" class="form-input" placeholder="Leave empty for auto" />
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="is_static" v-model="assignForm.is_static" class="rounded" />
            <label for="is_static" class="text-sm text-gray-700">Static assignment</label>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showAssignModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">Assign</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { useAuthStore } from '@/store/auth'
import { ipPoolsAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { PlusIcon, XMarkIcon, TrashIcon, ArrowPathIcon, MagnifyingGlassIcon, CircleStackIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const pools = ref([])
const viewingPool = ref(null)
const poolIPs = ref([])
const ipSearch = ref('')
const showCreateModal = ref(false)
const showAssignModal = ref(false)
const assignPool = ref(null)

const createForm = reactive({ name: '', network: '', gateway: '', dns1: '8.8.8.8', dns2: '8.8.4.4', description: '' })
const assignForm = reactive({ username: '', ip_address: '', is_static: false, pool_id: null })

const filteredIPs = computed(() => {
  if (!ipSearch.value) return poolIPs.value
  return poolIPs.value.filter(ip => ip.ip_address.includes(ipSearch.value) || (ip.username || '').includes(ipSearch.value))
})

function usagePct(pool) {
  if (!pool.total_ips) return 0
  return Math.round((pool.used_ips / pool.total_ips) * 100)
}

function formatDate(d) {
  try { return format(parseISO(d), 'MMM d, HH:mm') } catch { return d }
}

async function load() {
  loading.value = true
  try {
    const { data } = await ipPoolsAPI.list()
    pools.value = data.data || []
  } catch { /* silent */ } finally { loading.value = false }
}

async function viewIPs(pool) {
  viewingPool.value = pool
  try {
    const { data } = await ipPoolsAPI.listIPs(pool.id)
    poolIPs.value = data.data || []
  } catch { toast.error('Failed to load IPs') }
}

function openCreate() {
  Object.assign(createForm, { name: '', network: '', gateway: '', dns1: '8.8.8.8', dns2: '8.8.4.4', description: '' })
  showCreateModal.value = true
}

function openAssign(pool) {
  assignPool.value = pool
  assignForm.pool_id = pool.id
  assignForm.username = ''
  assignForm.ip_address = ''
  assignForm.is_static = false
  showAssignModal.value = true
}

async function createPool() {
  saving.value = true
  try {
    const { data } = await ipPoolsAPI.create(createForm)
    toast.success(data.message)
    showCreateModal.value = false
    load()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Create failed')
  } finally { saving.value = false }
}

async function assignIP() {
  saving.value = true
  try {
    const payload = { username: assignForm.username, pool_id: assignForm.pool_id, is_static: assignForm.is_static }
    if (assignForm.ip_address) payload.ip_address = assignForm.ip_address
    const { data } = await ipPoolsAPI.assign(payload)
    toast.success(data.message)
    showAssignModal.value = false
    load()
    if (viewingPool.value?.id === assignForm.pool_id) viewIPs(viewingPool.value)
  } catch (err) {
    toast.error(err.response?.data?.error || 'Assign failed')
  } finally { saving.value = false }
}

async function release(username) {
  if (!confirm(`Release IP assignment from "${username}"?`)) return
  try {
    await ipPoolsAPI.release({ username })
    toast.success('IP released')
    load()
    if (viewingPool.value) viewIPs(viewingPool.value)
  } catch { toast.error('Failed') }
}

async function removePool(pool) {
  if (!confirm(`Delete pool "${pool.name}" and all ${pool.used_ips} assignments?`)) return
  try {
    await ipPoolsAPI.delete(pool.id)
    toast.success('Pool deleted')
    if (viewingPool.value?.id === pool.id) viewingPool.value = null
    load()
  } catch { toast.error('Failed') }
}

onMounted(load)
</script>
