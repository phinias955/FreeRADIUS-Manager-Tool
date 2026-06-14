<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Vouchers</h1>
        <p class="text-sm text-gray-500 mt-0.5">Generate and manage prepaid access vouchers</p>
      </div>
      <div class="flex gap-2">
        <button @click="showGenModal = true" class="btn-primary">
          <PlusIcon class="w-4 h-4" />
          Generate Vouchers
        </button>
        <button @click="exportVouchers" class="btn-secondary">
          <ArrowDownTrayIcon class="w-4 h-4" />
          Export CSV
        </button>
      </div>
    </div>

    <!-- Summary cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-gray-900">{{ stats.total }}</p>
        <p class="text-xs text-gray-500 mt-1">Total</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-green-600">{{ stats.active }}</p>
        <p class="text-xs text-gray-500 mt-1">Active</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-blue-600">{{ stats.used }}</p>
        <p class="text-xs text-gray-500 mt-1">Used</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-red-500">{{ stats.disabled }}</p>
        <p class="text-xs text-gray-500 mt-1">Disabled</p>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap gap-3">
      <input v-model="filters.batch" type="text" class="form-input w-48" placeholder="Filter by batch..." @input="debouncedLoad" />
      <select v-model="filters.status" class="form-input w-36" @change="loadVouchers">
        <option value="">All Status</option>
        <option value="active">Active</option>
        <option value="used">Used</option>
        <option value="expired">Expired</option>
        <option value="disabled">Disabled</option>
      </select>
      <button v-if="selectedIds.length" @click="printSelected" class="btn-secondary ml-auto">
        <PrinterIcon class="w-4 h-4" />
        Print Selected ({{ selectedIds.length }})
      </button>
    </div>

    <!-- Table -->
    <div class="card p-0 overflow-hidden">
      <div class="table-container rounded-none border-0">
        <table class="table">
          <thead>
            <tr>
              <th class="w-8">
                <input type="checkbox" @change="toggleAll" :checked="allSelected" class="rounded" />
              </th>
              <th>Code</th>
              <th>Batch</th>
              <th>Status</th>
              <th>Data Limit</th>
              <th>Time Limit</th>
              <th>Expires</th>
              <th>Redeemed By</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="10" class="text-center py-12 text-gray-400">Loading…</td>
            </tr>
            <tr v-else-if="!vouchers.length">
              <td colspan="10" class="text-center py-12 text-gray-400">No vouchers found. Generate some to get started.</td>
            </tr>
            <tr v-for="v in vouchers" :key="v.id">
              <td>
                <input type="checkbox" :value="v.id" v-model="selectedIds" class="rounded" />
              </td>
              <td>
                <span class="font-mono text-sm font-semibold tracking-wider text-gray-900">{{ v.code }}</span>
              </td>
              <td class="text-gray-500 text-sm">{{ v.batch_name || '—' }}</td>
              <td>
                <span class="badge" :class="statusClass(v.status)">{{ v.status }}</span>
              </td>
              <td class="text-sm text-gray-600">{{ v.data_limit_mb ? v.data_limit_mb + ' MB' : 'Unlimited' }}</td>
              <td class="text-sm text-gray-600">{{ v.time_limit_minutes ? v.time_limit_minutes + ' min' : 'Unlimited' }}</td>
              <td class="text-xs text-gray-500">{{ formatDate(v.expires_at) }}</td>
              <td class="text-sm text-gray-500">{{ v.redeemed_by || '—' }}</td>
              <td class="text-xs text-gray-400">{{ formatDate(v.created_at) }}</td>
              <td>
                <div class="flex gap-1">
                  <button v-if="v.status === 'active'" @click="disable(v)" class="text-xs px-2 py-1 rounded bg-yellow-100 text-yellow-700 hover:bg-yellow-200">Disable</button>
                  <button @click="remove(v)" class="text-xs px-2 py-1 rounded bg-red-100 text-red-600 hover:bg-red-200">Delete</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-gray-500">
      <span>{{ total }} total vouchers</span>
      <div class="flex gap-2">
        <button @click="page--; loadVouchers()" :disabled="page <= 1" class="btn-secondary px-3 py-1 text-xs">Prev</button>
        <span class="px-2 py-1">Page {{ page }}</span>
        <button @click="page++; loadVouchers()" :disabled="page * limit >= total" class="btn-secondary px-3 py-1 text-xs">Next</button>
      </div>
    </div>

    <!-- Generate Modal -->
    <div v-if="showGenModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">Generate Vouchers</h3>
          <button @click="showGenModal = false" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="generate" class="p-6 space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Count <span class="text-red-500">*</span></label>
              <input v-model.number="genForm.count" type="number" min="1" max="500" class="form-input" required />
            </div>
            <div>
              <label class="form-label">Valid Days</label>
              <input v-model.number="genForm.valid_days" type="number" min="1" class="form-input" placeholder="30" />
            </div>
          </div>
          <div>
            <label class="form-label">Batch Name</label>
            <input v-model="genForm.batch_name" type="text" class="form-input" placeholder="e.g. Hotel-Room-June" />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Data Limit (MB)</label>
              <input v-model.number="genForm.data_limit_mb" type="number" min="0" class="form-input" placeholder="Leave empty = unlimited" />
            </div>
            <div>
              <label class="form-label">Time Limit (min)</label>
              <input v-model.number="genForm.time_limit_minutes" type="number" min="0" class="form-input" placeholder="Leave empty = unlimited" />
            </div>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showGenModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="generating">
              <span v-if="generating" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Generate {{ genForm.count || '' }} Vouchers
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Print Frame (hidden) -->
    <div id="print-area" class="hidden"></div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { voucherAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { PlusIcon, ArrowDownTrayIcon, PrinterIcon, XMarkIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const loading = ref(false)
const generating = ref(false)
const showGenModal = ref(false)
const vouchers = ref([])
const total = ref(0)
const page = ref(1)
const limit = 50
const selectedIds = ref([])
let searchTimer

const filters = reactive({ batch: '', status: '' })
const genForm = reactive({ count: 10, batch_name: '', data_limit_mb: null, time_limit_minutes: null, valid_days: 30 })

const stats = computed(() => ({
  total: total.value,
  active: vouchers.value.filter(v => v.status === 'active').length,
  used: vouchers.value.filter(v => v.status === 'used').length,
  disabled: vouchers.value.filter(v => v.status === 'disabled').length,
}))

const allSelected = computed(() => vouchers.value.length > 0 && selectedIds.value.length === vouchers.value.length)

function toggleAll(e) {
  selectedIds.value = e.target.checked ? vouchers.value.map(v => v.id) : []
}

function statusClass(status) {
  return {
    active: 'badge-green',
    used: 'badge-gray',
    expired: 'badge-yellow',
    disabled: 'badge-red',
  }[status] || 'badge-gray'
}

function formatDate(d) {
  if (!d) return '—'
  try { return format(parseISO(d), 'MMM d, yyyy') } catch { return d }
}

async function loadVouchers() {
  loading.value = true
  try {
    const { data } = await voucherAPI.list({ page: page.value, limit, ...filters })
    vouchers.value = data.data || []
    total.value = data.total || 0
  } catch { /* silent */ } finally {
    loading.value = false
  }
}

function debouncedLoad() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(loadVouchers, 400)
}

async function generate() {
  generating.value = true
  try {
    const payload = { ...genForm }
    if (!payload.data_limit_mb) delete payload.data_limit_mb
    if (!payload.time_limit_minutes) delete payload.time_limit_minutes
    if (!payload.valid_days) payload.valid_days = 30
    const { data } = await voucherAPI.generate(payload)
    toast.success(data.message)
    showGenModal.value = false
    Object.assign(genForm, { count: 10, batch_name: '', data_limit_mb: null, time_limit_minutes: null, valid_days: 30 })
    loadVouchers()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Generation failed')
  } finally {
    generating.value = false
  }
}

async function disable(v) {
  if (!confirm(`Disable voucher ${v.code}?`)) return
  try {
    await voucherAPI.disable(v.id)
    toast.success('Voucher disabled')
    loadVouchers()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed')
  }
}

async function remove(v) {
  if (!confirm(`Delete voucher ${v.code}? This cannot be undone.`)) return
  try {
    await voucherAPI.delete(v.id)
    toast.success('Voucher deleted')
    loadVouchers()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed')
  }
}

async function exportVouchers() {
  try {
    const { data } = await voucherAPI.export({ batch: filters.batch })
    const url = URL.createObjectURL(new Blob([data]))
    const a = document.createElement('a')
    a.href = url
    a.download = `vouchers_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  } catch { toast.error('Export failed') }
}

function printSelected() {
  const selected = vouchers.value.filter(v => selectedIds.value.includes(v.id))
  const html = `
    <html><head><title>Vouchers</title>
    <style>
      body { font-family: monospace; }
      .grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; padding: 12px; }
      .card { border: 2px dashed #999; border-radius: 8px; padding: 14px; text-align: center; page-break-inside: avoid; }
      .code { font-size: 18px; font-weight: bold; letter-spacing: 3px; margin: 8px 0; }
      .info { font-size: 11px; color: #555; }
      @media print { body { margin: 0; } }
    </style></head>
    <body><div class="grid">
      ${selected.map(v => `
        <div class="card">
          <div class="info">WiFi Access Voucher</div>
          <div class="code">${v.code}</div>
          <div class="info">${v.data_limit_mb ? v.data_limit_mb + ' MB data' : ''}${v.time_limit_minutes ? ' · ' + v.time_limit_minutes + ' min' : ''}</div>
          <div class="info">Expires: ${formatDate(v.expires_at)}</div>
        </div>`).join('')}
    </div></body></html>`
  const win = window.open('', '_blank')
  win.document.write(html)
  win.document.close()
  win.print()
}

onMounted(loadVouchers)
</script>
