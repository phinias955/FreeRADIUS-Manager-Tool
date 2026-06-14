<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Billing</h1>
        <p class="text-sm text-gray-500 mt-0.5">Invoices and payment tracking</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
        <PlusIcon class="w-4 h-4" />
        New Invoice
      </button>
    </div>

    <!-- Summary row -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <div class="card p-4">
        <p class="text-xs text-gray-500">Total Revenue</p>
        <p class="text-2xl font-bold text-green-600 mt-1">{{ summary.currency }} {{ summary.totalRevenue?.toFixed(2) }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs text-gray-500">Paid</p>
        <p class="text-2xl font-bold text-green-600 mt-1">{{ summary.paid }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs text-gray-500">Pending</p>
        <p class="text-2xl font-bold text-yellow-600 mt-1">{{ summary.pending }}</p>
      </div>
      <div class="card p-4">
        <p class="text-xs text-gray-500">Overdue</p>
        <p class="text-2xl font-bold text-red-600 mt-1">{{ summary.overdue }}</p>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap gap-3">
      <input v-model="filters.username" type="text" class="form-input w-48" placeholder="Search username..." @input="debouncedLoad" />
      <select v-model="filters.status" @change="loadInvoices" class="form-input w-36">
        <option value="">All Status</option>
        <option value="pending">Pending</option>
        <option value="paid">Paid</option>
        <option value="overdue">Overdue</option>
        <option value="cancelled">Cancelled</option>
      </select>
    </div>

    <!-- Table -->
    <div class="card p-0 overflow-hidden">
      <div class="table-container rounded-none border-0">
        <table class="table">
          <thead>
            <tr>
              <th>Invoice #</th>
              <th>Username</th>
              <th>Plan</th>
              <th>Amount</th>
              <th>Status</th>
              <th>Due Date</th>
              <th>Paid At</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="8" class="text-center py-12 text-gray-400">Loading…</td>
            </tr>
            <tr v-else-if="!invoices.length">
              <td colspan="8" class="text-center py-12 text-gray-400">No invoices found</td>
            </tr>
            <tr v-for="inv in invoices" :key="inv.id">
              <td class="font-mono text-xs text-gray-700">{{ inv.invoice_number }}</td>
              <td class="font-medium">{{ inv.username }}</td>
              <td class="text-gray-500 text-sm">{{ inv.plan_name || '—' }}</td>
              <td class="font-semibold">{{ inv.currency }} {{ inv.amount?.toFixed(2) }}</td>
              <td>
                <span class="badge" :class="statusClass(inv.status)">{{ inv.status }}</span>
              </td>
              <td class="text-xs text-gray-500">{{ inv.due_date || '—' }}</td>
              <td class="text-xs text-gray-500">{{ inv.paid_at ? formatDate(inv.paid_at) : '—' }}</td>
              <td>
                <div class="flex gap-1" v-if="authStore.isAdmin">
                  <button v-if="inv.status === 'pending'" @click="markPaid(inv)" class="text-xs px-2 py-1 rounded bg-green-100 text-green-700 hover:bg-green-200">Mark Paid</button>
                  <button v-if="inv.status === 'pending'" @click="markOverdue(inv)" class="text-xs px-2 py-1 rounded bg-orange-100 text-orange-700 hover:bg-orange-200">Overdue</button>
                  <button @click="remove(inv)" class="text-xs px-2 py-1 rounded bg-red-100 text-red-600 hover:bg-red-200">Del</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-gray-500">
      <span>{{ total }} total invoices</span>
      <div class="flex gap-2">
        <button @click="page--; loadInvoices()" :disabled="page <= 1" class="btn-secondary px-3 py-1 text-xs">Prev</button>
        <span class="px-2 py-1">Page {{ page }}</span>
        <button @click="page++; loadInvoices()" :disabled="page * 20 >= total" class="btn-secondary px-3 py-1 text-xs">Next</button>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">New Invoice</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="create" class="p-6 space-y-4">
          <div>
            <label class="form-label">Username <span class="text-red-500">*</span></label>
            <input v-model="form.username" type="text" class="form-input" required />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Amount</label>
              <input v-model.number="form.amount" type="number" min="0" step="0.01" class="form-input" placeholder="0.00" />
            </div>
            <div>
              <label class="form-label">Currency</label>
              <input v-model="form.currency" type="text" class="form-input" maxlength="3" placeholder="USD" />
            </div>
          </div>
          <div>
            <label class="form-label">Due Date</label>
            <input v-model="form.due_date" type="date" class="form-input" />
          </div>
          <div>
            <label class="form-label">Notes</label>
            <input v-model="form.notes" type="text" class="form-input" placeholder="Optional notes" />
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Create Invoice
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
import { billingAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { PlusIcon, XMarkIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const invoices = ref([])
const total = ref(0)
const page = ref(1)
let searchTimer

const summary = reactive({ totalRevenue: 0, paid: 0, pending: 0, overdue: 0, currency: 'USD' })
const filters = reactive({ username: '', status: '' })
const form = reactive({ username: '', amount: 0, currency: 'USD', due_date: '', notes: '' })

function statusClass(s) {
  return { paid: 'badge-green', pending: 'badge-yellow', overdue: 'badge-red', cancelled: 'badge-gray' }[s] || 'badge-gray'
}

function formatDate(d) {
  if (!d) return '—'
  try { return format(parseISO(d), 'MMM d, yyyy HH:mm') } catch { return d }
}

async function loadInvoices() {
  loading.value = true
  try {
    const { data } = await billingAPI.list({ page: page.value, ...filters })
    invoices.value = data.data || []
    total.value = data.total || 0
    if (data.summary) {
      Object.assign(summary, data.summary)
    }
  } catch { /* silent */ } finally { loading.value = false }
}

function debouncedLoad() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(loadInvoices, 400)
}

async function create() {
  saving.value = true
  try {
    await billingAPI.create(form)
    toast.success('Invoice created')
    showModal.value = false
    Object.assign(form, { username: '', amount: 0, currency: 'USD', due_date: '', notes: '' })
    loadInvoices()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed')
  } finally { saving.value = false }
}

function openCreate() { showModal.value = true }

async function markPaid(inv) {
  try {
    await billingAPI.update(inv.id, { status: 'paid' })
    toast.success('Marked as paid')
    loadInvoices()
  } catch (err) { toast.error('Failed') }
}

async function markOverdue(inv) {
  try {
    await billingAPI.update(inv.id, { status: 'overdue' })
    loadInvoices()
  } catch { /* silent */ }
}

async function remove(inv) {
  if (!confirm(`Delete invoice ${inv.invoice_number}?`)) return
  try {
    await billingAPI.delete(inv.id)
    toast.success('Deleted')
    loadInvoices()
  } catch (err) { toast.error('Failed') }
}

onMounted(loadInvoices)
</script>
