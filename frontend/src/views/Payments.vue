<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Payments</h1>
        <p class="text-sm text-gray-500 mt-0.5">Track all customer payments and generate receipts</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
        <PlusIcon class="w-4 h-4" />
        Record Payment
      </button>
    </div>

    <!-- Summary cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <div class="card p-4 flex items-center gap-3">
        <div class="w-9 h-9 bg-green-100 rounded-xl flex items-center justify-center">
          <CurrencyDollarIcon class="w-5 h-5 text-green-600" />
        </div>
        <div>
          <p class="text-xs text-gray-500">This Month</p>
          <p class="text-xl font-bold text-green-600">${{ summary.total_month?.toFixed(2) || '0.00' }}</p>
        </div>
      </div>
      <div class="card p-4 flex items-center gap-3">
        <div class="w-9 h-9 bg-blue-100 rounded-xl flex items-center justify-center">
          <BanknotesIcon class="w-5 h-5 text-blue-600" />
        </div>
        <div>
          <p class="text-xs text-gray-500">All Time</p>
          <p class="text-xl font-bold text-blue-600">${{ summary.total_all?.toFixed(2) || '0.00' }}</p>
        </div>
      </div>
      <div class="card p-4 flex items-center gap-3">
        <div class="w-9 h-9 bg-purple-100 rounded-xl flex items-center justify-center">
          <ReceiptPercentIcon class="w-5 h-5 text-purple-600" />
        </div>
        <div>
          <p class="text-xs text-gray-500">This Month (count)</p>
          <p class="text-xl font-bold text-purple-600">{{ summary.count_month || 0 }}</p>
        </div>
      </div>
      <div class="card p-4 flex items-center gap-3">
        <div class="w-9 h-9 bg-orange-100 rounded-xl flex items-center justify-center">
          <ReceiptPercentIcon class="w-5 h-5 text-orange-600" />
        </div>
        <div>
          <p class="text-xs text-gray-500">Total (count)</p>
          <p class="text-xl font-bold text-orange-600">{{ summary.count_all || 0 }}</p>
        </div>
      </div>
    </div>

    <!-- Method breakdown -->
    <div v-if="summary.by_method?.length" class="card">
      <h3 class="font-semibold text-gray-900 mb-3 text-sm">Revenue by Payment Method</h3>
      <div class="flex flex-wrap gap-3">
        <div v-for="m in summary.by_method" :key="m.method"
          class="flex items-center gap-2 bg-gray-50 rounded-xl px-3 py-2">
          <span class="text-xs font-medium capitalize">{{ m.method.replace('_',' ') }}</span>
          <span class="text-sm font-bold text-blue-700">${{ m.total?.toFixed(2) }}</span>
          <span class="text-xs text-gray-400">({{ m.count }})</span>
        </div>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex items-center gap-3 flex-wrap">
      <input v-model="filterUsername" type="text" class="form-input w-48 text-sm" placeholder="Filter by username…" @input="debouncedLoad" />
      <select v-model="filterMethod" class="form-input w-36 text-sm" @change="load">
        <option value="">All methods</option>
        <option value="cash">Cash</option>
        <option value="bank_transfer">Bank Transfer</option>
        <option value="mobile_money">Mobile Money</option>
        <option value="card">Card</option>
        <option value="online">Online</option>
      </select>
    </div>

    <!-- Payments table -->
    <div class="card p-0 overflow-hidden">
      <table class="table">
        <thead>
          <tr>
            <th>Receipt</th>
            <th>Customer</th>
            <th>Invoice</th>
            <th>Amount</th>
            <th>Method</th>
            <th>Date</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="7" class="text-center py-8 text-gray-400">Loading…</td></tr>
          <tr v-else-if="!payments.length"><td colspan="7" class="text-center py-8 text-gray-400">No payments found</td></tr>
          <tr v-for="p in payments" :key="p.id">
            <td><span class="font-mono text-xs">{{ p.receipt_number }}</span></td>
            <td class="text-sm font-medium">{{ p.username || '—' }}</td>
            <td class="text-xs text-gray-500">{{ p.invoice_number || '—' }}</td>
            <td>
              <span class="font-bold text-green-600">{{ p.currency }} {{ p.amount?.toFixed(2) }}</span>
            </td>
            <td>
              <span class="badge badge-blue text-xs capitalize">{{ p.payment_method?.replace('_',' ') }}</span>
            </td>
            <td class="text-xs text-gray-400">{{ formatDate(p.created_at) }}</td>
            <td>
              <div class="flex gap-1">
                <a :href="`/api/v1/payments/${p.id}/receipt`" target="_blank"
                  class="p-1.5 rounded hover:bg-blue-50 text-gray-400 hover:text-blue-600" title="View Receipt">
                  <ReceiptPercentIcon class="w-4 h-4" />
                </a>
                <button v-if="authStore.isSuperAdmin" @click="deletePayment(p)"
                  class="p-1.5 rounded hover:bg-red-50 text-gray-400 hover:text-red-500">
                  <TrashIcon class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Record Payment Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">Record Payment</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Username</label>
              <input v-model="form.username" type="text" class="form-input" placeholder="RADIUS username" />
            </div>
            <div>
              <label class="form-label">Amount <span class="text-red-500">*</span></label>
              <input v-model.number="form.amount" type="number" step="0.01" min="0.01" class="form-input" required />
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Currency</label>
              <input v-model="form.currency" type="text" class="form-input" placeholder="USD" />
            </div>
            <div>
              <label class="form-label">Payment Method</label>
              <select v-model="form.payment_method" class="form-input">
                <option value="cash">Cash</option>
                <option value="bank_transfer">Bank Transfer</option>
                <option value="mobile_money">Mobile Money</option>
                <option value="card">Card</option>
                <option value="online">Online</option>
              </select>
            </div>
          </div>
          <div>
            <label class="form-label">Gateway Reference</label>
            <input v-model="form.gateway_ref" type="text" class="form-input" placeholder="Transaction ID, MPESA ref, etc." />
          </div>
          <div>
            <label class="form-label">Notes</label>
            <input v-model="form.notes" type="text" class="form-input" placeholder="Optional note" />
          </div>
          <div v-if="lastReceipt" class="p-3 bg-green-50 rounded-xl text-sm text-green-700 flex items-center justify-between">
            <span>Receipt: <strong>{{ lastReceipt }}</strong></span>
            <a :href="`/api/v1/payments/${lastPaymentId}/receipt`" target="_blank"
              class="text-xs px-3 py-1 bg-green-600 text-white rounded-lg">View</a>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Close</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Record & Generate Receipt
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { useAuthStore } from '@/store/auth'
import { paymentsAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { PlusIcon, XMarkIcon, TrashIcon, CurrencyDollarIcon, BanknotesIcon, ReceiptPercentIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const payments = ref([])
const summary = ref({})
const filterUsername = ref('')
const filterMethod = ref('')
const lastReceipt = ref('')
const lastPaymentId = ref(null)
let debounceTimer

const form = reactive({ username: '', amount: '', currency: 'USD', payment_method: 'cash', gateway_ref: '', notes: '' })

function formatDate(d) { try { return format(parseISO(d), 'MMM d, yyyy HH:mm') } catch { return d } }

async function load() {
  loading.value = true
  try {
    const [r1, r2] = await Promise.all([
      paymentsAPI.list({ username: filterUsername.value, method: filterMethod.value }),
      paymentsAPI.summary()
    ])
    payments.value = r1.data.data || []
    summary.value = r2.data
  } catch { } finally { loading.value = false }
}

function debouncedLoad() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(load, 300)
}

function openCreate() {
  Object.assign(form, { username: '', amount: '', currency: 'USD', payment_method: 'cash', gateway_ref: '', notes: '' })
  lastReceipt.value = ''
  lastPaymentId.value = null
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    const { data } = await paymentsAPI.create(form)
    lastReceipt.value = data.receipt_number
    lastPaymentId.value = data.id
    toast.success(`Payment recorded — Receipt: ${data.receipt_number}`)
    Object.assign(form, { username: '', amount: '', currency: 'USD', payment_method: 'cash', gateway_ref: '', notes: '' })
    load()
  } catch (err) { toast.error(err.response?.data?.error || 'Failed') }
  finally { saving.value = false }
}

async function deletePayment(p) {
  if (!confirm(`Delete payment ${p.receipt_number}?`)) return
  try { await paymentsAPI.delete(p.id); toast.success('Deleted'); load() }
  catch { toast.error('Failed') }
}

onMounted(load)
</script>
