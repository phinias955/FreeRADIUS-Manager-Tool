<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Customers</h1>
        <p class="text-sm text-gray-500 mt-0.5">CRM — manage customer profiles and contracts</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
        <PlusIcon class="w-4 h-4" />
        New Customer
      </button>
    </div>

    <!-- Search -->
    <div class="card p-4">
      <input v-model="search" type="text" class="form-input" placeholder="Search by name, email, phone, or username…"
        @input="debouncedLoad" />
    </div>

    <!-- Table -->
    <div class="card p-0 overflow-hidden">
      <table class="table">
        <thead>
          <tr>
            <th>Customer</th>
            <th>Username</th>
            <th>Contact</th>
            <th>Contract</th>
            <th>Tickets</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="6" class="text-center py-8 text-gray-400">Loading…</td></tr>
          <tr v-else-if="!customers.length"><td colspan="6" class="text-center py-8 text-gray-400">No customers found</td></tr>
          <tr v-for="c in customers" :key="c.id" class="cursor-pointer hover:bg-gray-50" @click="viewCustomer(c)">
            <td>
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 rounded-full bg-gradient-to-br from-blue-400 to-purple-500 flex items-center justify-center text-white text-sm font-semibold flex-shrink-0">
                  {{ initials(c.full_name) }}
                </div>
                <div>
                  <p class="font-medium text-gray-900 text-sm">{{ c.full_name || '—' }}</p>
                  <p class="text-xs text-gray-400">{{ c.city || c.country }}</p>
                </div>
              </div>
            </td>
            <td><span class="font-mono text-xs">{{ c.username || '—' }}</span></td>
            <td>
              <p class="text-sm">{{ c.phone || '—' }}</p>
              <p class="text-xs text-gray-400">{{ c.email || '' }}</p>
            </td>
            <td>
              <div class="text-xs">
                <p v-if="c.contract_end">Ends {{ c.contract_end }}</p>
                <p v-else class="text-gray-400">No contract</p>
              </div>
            </td>
            <td>
              <span v-if="c.open_tickets > 0" class="badge badge-red">{{ c.open_tickets }} open</span>
              <span v-else class="badge badge-green">OK</span>
            </td>
            <td @click.stop>
              <div class="flex gap-1">
                <button @click="openEdit(c)" class="p-1.5 rounded hover:bg-gray-100 text-gray-400 hover:text-blue-600">
                  <PencilIcon class="w-4 h-4" />
                </button>
                <button v-if="authStore.isAdmin" @click="removeCustomer(c)" class="p-1.5 rounded hover:bg-red-50 text-gray-400 hover:text-red-500">
                  <TrashIcon class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Customer detail side-panel -->
    <div v-if="selected" class="card">
      <div class="flex items-start justify-between mb-4">
        <div class="flex items-center gap-3">
          <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-400 to-purple-500 flex items-center justify-center text-white text-lg font-bold">
            {{ initials(selected.customer?.full_name) }}
          </div>
          <div>
            <h3 class="font-semibold text-gray-900">{{ selected.customer?.full_name }}</h3>
            <p class="text-sm text-gray-500">{{ selected.customer?.username ? '@' + selected.customer.username : 'No linked account' }}</p>
          </div>
        </div>
        <button @click="selected = null" class="btn-secondary text-xs py-1.5">Close</button>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm mb-4">
        <div><p class="text-xs text-gray-400">Email</p><p class="font-medium">{{ selected.customer?.email || '—' }}</p></div>
        <div><p class="text-xs text-gray-400">Phone</p><p class="font-medium">{{ selected.customer?.phone || '—' }}</p></div>
        <div><p class="text-xs text-gray-400">ID Number</p><p class="font-medium">{{ selected.customer?.id_number || '—' }}</p></div>
        <div><p class="text-xs text-gray-400">Address</p><p class="font-medium">{{ selected.customer?.address || '—' }}</p></div>
        <div><p class="text-xs text-gray-400">Contract</p><p class="font-medium">{{ selected.customer?.contract_start || '—' }} → {{ selected.customer?.contract_end || '∞' }}</p></div>
        <div><p class="text-xs text-gray-400">Revenue</p><p class="font-medium text-green-600">${{ selected.usage?.paid_total?.toFixed(2) || '0.00' }}</p></div>
      </div>
      <div v-if="selected.customer?.notes" class="p-3 bg-yellow-50 rounded-xl text-sm text-yellow-800 mb-4">
        {{ selected.customer.notes }}
      </div>
      <!-- Tickets -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <h4 class="text-sm font-medium text-gray-700">Support Tickets</h4>
          <button @click="newTicketFor = selected.customer; showTicketModal = true" class="btn-secondary text-xs py-1">
            <PlusIcon class="w-3 h-3" /> New Ticket
          </button>
        </div>
        <div v-if="!selected.tickets?.length" class="text-xs text-gray-400 py-2">No tickets</div>
        <div v-for="t in selected.tickets" :key="t.id"
          class="flex items-center justify-between p-2 rounded-lg border border-gray-100 mb-1.5">
          <div>
            <p class="text-sm font-medium">{{ t.title }}</p>
            <p class="text-xs text-gray-400">{{ t.created_at?.slice(0,10) }}</p>
          </div>
          <div class="flex items-center gap-2">
            <span class="badge" :class="priorityBadge(t.priority)">{{ t.priority }}</span>
            <span class="badge" :class="statusBadge(t.status)">{{ t.status }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between p-6 border-b sticky top-0 bg-white z-10">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Customer</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="form-label">Full Name <span class="text-red-500">*</span></label>
              <input v-model="form.full_name" type="text" class="form-input" required placeholder="John Doe" />
            </div>
            <div>
              <label class="form-label">Username (RADIUS)</label>
              <input v-model="form.username" type="text" class="form-input" placeholder="Link to RADIUS user" />
            </div>
            <div>
              <label class="form-label">ID Number</label>
              <input v-model="form.id_number" type="text" class="form-input" placeholder="National ID" />
            </div>
            <div>
              <label class="form-label">Email</label>
              <input v-model="form.email" type="email" class="form-input" />
            </div>
            <div>
              <label class="form-label">Phone</label>
              <input v-model="form.phone" type="tel" class="form-input" placeholder="+263..." />
            </div>
            <div>
              <label class="form-label">City</label>
              <input v-model="form.city" type="text" class="form-input" />
            </div>
            <div>
              <label class="form-label">Country</label>
              <input v-model="form.country" type="text" class="form-input" />
            </div>
            <div>
              <label class="form-label">Contract Start</label>
              <input v-model="form.contract_start" type="date" class="form-input" />
            </div>
            <div>
              <label class="form-label">Contract End</label>
              <input v-model="form.contract_end" type="date" class="form-input" />
            </div>
          </div>
          <div>
            <label class="form-label">Address</label>
            <input v-model="form.address" type="text" class="form-input" />
          </div>
          <div>
            <label class="form-label">Notes</label>
            <textarea v-model="form.notes" rows="2" class="form-input" placeholder="Internal notes…"></textarea>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              {{ editing ? 'Update' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Quick ticket modal -->
    <div v-if="showTicketModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">New Ticket for {{ newTicketFor?.full_name }}</h3>
          <button @click="showTicketModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="createQuickTicket" class="p-6 space-y-4">
          <div>
            <label class="form-label">Title <span class="text-red-500">*</span></label>
            <input v-model="ticketForm.title" type="text" class="form-input" required placeholder="Describe the issue…" />
          </div>
          <div>
            <label class="form-label">Priority</label>
            <select v-model="ticketForm.priority" class="form-input">
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="urgent">Urgent</option>
            </select>
          </div>
          <div>
            <label class="form-label">Description</label>
            <textarea v-model="ticketForm.description" rows="3" class="form-input"></textarea>
          </div>
          <div class="flex justify-end gap-3">
            <button type="button" @click="showTicketModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary">Create Ticket</button>
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
import { customersAPI, ticketsAPI } from '@/api'
import { PlusIcon, XMarkIcon, TrashIcon, PencilIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const showTicketModal = ref(false)
const editing = ref(null)
const search = ref('')
const customers = ref([])
const selected = ref(null)
const newTicketFor = ref(null)
let debounceTimer

const form = reactive({ username: '', full_name: '', email: '', phone: '', id_number: '', address: '', city: '', country: 'Zimbabwe', contract_start: '', contract_end: '', notes: '' })
const ticketForm = reactive({ title: '', priority: 'medium', description: '' })

function initials(name) {
  if (!name) return '?'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
}
function priorityBadge(p) { return { urgent: 'badge-red', high: 'badge-orange', medium: 'badge-blue', low: 'badge-gray' }[p] || 'badge-gray' }
function statusBadge(s) { return { open: 'badge-red', in_progress: 'badge-blue', resolved: 'badge-green', closed: 'badge-gray' }[s] || 'badge-gray' }

async function load() {
  loading.value = true
  try {
    const { data } = await customersAPI.list({ search: search.value, limit: 50 })
    customers.value = data.data || []
  } catch { } finally { loading.value = false }
}

function debouncedLoad() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(load, 300)
}

async function viewCustomer(c) {
  try {
    const { data } = await customersAPI.get(c.id)
    selected.value = data
  } catch { selected.value = { customer: c, tickets: [], usage: {} } }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { username: '', full_name: '', email: '', phone: '', id_number: '', address: '', city: '', country: 'Zimbabwe', contract_start: '', contract_end: '', notes: '' })
  showModal.value = true
}

function openEdit(c) {
  editing.value = c.id
  Object.assign(form, { username: c.username || '', full_name: c.full_name || '', email: c.email || '', phone: c.phone || '', id_number: c.id_number || '', address: c.address || '', city: c.city || '', country: c.country || 'Zimbabwe', contract_start: c.contract_start || '', contract_end: c.contract_end || '', notes: c.notes || '' })
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) { await customersAPI.update(editing.value, form); toast.success('Updated') }
    else { await customersAPI.create(form); toast.success('Created') }
    showModal.value = false; load()
  } catch (err) { toast.error(err.response?.data?.error || 'Save failed') }
  finally { saving.value = false }
}

async function removeCustomer(c) {
  if (!confirm(`Delete customer "${c.full_name}"?`)) return
  try { await customersAPI.delete(c.id); toast.success('Deleted'); load() }
  catch { toast.error('Failed') }
}

async function createQuickTicket() {
  try {
    await ticketsAPI.create({ ...ticketForm, customer_id: newTicketFor.value?.id, username: newTicketFor.value?.username })
    toast.success('Ticket created')
    showTicketModal.value = false
    ticketForm.title = ''; ticketForm.priority = 'medium'; ticketForm.description = ''
    if (selected.value) viewCustomer(selected.value.customer)
  } catch { toast.error('Failed') }
}

onMounted(load)
</script>
