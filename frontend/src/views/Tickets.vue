<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Support Tickets</h1>
        <p class="text-sm text-gray-500 mt-0.5">Track and resolve customer issues</p>
      </div>
      <button @click="openCreate" class="btn-primary">
        <PlusIcon class="w-4 h-4" />
        New Ticket
      </button>
    </div>

    <!-- Status cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <div v-for="s in statusCards" :key="s.key"
        class="card p-4 cursor-pointer transition-all"
        :class="filterStatus === s.key ? 'ring-2 ring-blue-500' : ''"
        @click="filterStatus = filterStatus === s.key ? '' : s.key; load()">
        <p class="text-2xl font-bold" :class="s.color">{{ counts[s.countKey] || 0 }}</p>
        <p class="text-xs text-gray-500 mt-0.5">{{ s.label }}</p>
      </div>
    </div>

    <!-- Priority filter -->
    <div class="flex items-center gap-2 flex-wrap">
      <span class="text-xs text-gray-500">Priority:</span>
      <button v-for="p in ['', 'urgent', 'high', 'medium', 'low']" :key="p"
        @click="filterPriority = p; load()"
        class="text-xs px-3 py-1 rounded-lg border transition-colors"
        :class="filterPriority === p ? 'bg-blue-600 border-blue-600 text-white' : 'border-gray-200 text-gray-600 hover:bg-gray-50'">
        {{ p || 'All' }}
      </button>
    </div>

    <!-- Tickets board -->
    <div class="card p-0 overflow-hidden">
      <table class="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Title</th>
            <th>Customer</th>
            <th>Priority</th>
            <th>Status</th>
            <th>Assigned</th>
            <th>Created</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="8" class="text-center py-8 text-gray-400">Loading…</td></tr>
          <tr v-else-if="!tickets.length"><td colspan="8" class="text-center py-8 text-gray-400">No tickets found</td></tr>
          <tr v-for="t in tickets" :key="t.id">
            <td class="font-mono text-xs text-gray-400">#{{ t.id }}</td>
            <td>
              <p class="font-medium text-sm text-gray-900">{{ t.title }}</p>
              <p v-if="t.username" class="text-xs text-gray-400 font-mono">{{ t.username }}</p>
            </td>
            <td class="text-sm">{{ t.customer_name || '—' }}</td>
            <td>
              <span class="badge" :class="priorityBadge(t.priority)">{{ t.priority }}</span>
            </td>
            <td>
              <select :value="t.status" @change="updateStatus(t, $event.target.value)"
                class="text-xs rounded-lg border border-gray-200 px-2 py-1 cursor-pointer"
                :class="statusColor(t.status)">
                <option value="open">open</option>
                <option value="in_progress">in_progress</option>
                <option value="resolved">resolved</option>
                <option value="closed">closed</option>
              </select>
            </td>
            <td class="text-xs text-gray-500">{{ t.assignee_name || '—' }}</td>
            <td class="text-xs text-gray-400">{{ t.created_at?.slice(0,10) }}</td>
            <td>
              <button @click="deleteTicket(t)" class="p-1.5 rounded hover:bg-red-50 text-gray-400 hover:text-red-500">
                <TrashIcon class="w-3.5 h-3.5" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">New Ticket</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div>
            <label class="form-label">Title <span class="text-red-500">*</span></label>
            <input v-model="form.title" type="text" class="form-input" required placeholder="Short summary of the issue" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Username (RADIUS)</label>
              <input v-model="form.username" type="text" class="form-input" placeholder="Affected user" />
            </div>
            <div>
              <label class="form-label">Priority</label>
              <select v-model="form.priority" class="form-input">
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="urgent">Urgent</option>
              </select>
            </div>
          </div>
          <div>
            <label class="form-label">Description</label>
            <textarea v-model="form.description" rows="4" class="form-input" placeholder="Full description of the issue…"></textarea>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Create Ticket
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
import { ticketsAPI } from '@/api'
import { PlusIcon, XMarkIcon, TrashIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const filterStatus = ref('')
const filterPriority = ref('')
const tickets = ref([])
const counts = ref({})
const form = reactive({ title: '', username: '', priority: 'medium', description: '' })

const statusCards = [
  { key: 'open', label: 'Open', countKey: 'open', color: 'text-red-600' },
  { key: 'in_progress', label: 'In Progress', countKey: 'in_progress', color: 'text-blue-600' },
  { key: 'resolved', label: 'Resolved', countKey: 'resolved', color: 'text-green-600' },
  { key: 'closed', label: 'Closed', countKey: 'closed', color: 'text-gray-500' },
]

function priorityBadge(p) { return { urgent: 'badge-red', high: 'badge-orange', medium: 'badge-blue', low: 'badge-gray' }[p] || 'badge-gray' }
function statusColor(s) { return { open: 'text-red-600', in_progress: 'text-blue-600', resolved: 'text-green-600', closed: 'text-gray-500' }[s] || '' }

async function load() {
  loading.value = true
  try {
    const { data } = await ticketsAPI.list({ status: filterStatus.value, priority: filterPriority.value, limit: 100 })
    tickets.value = data.data || []
    counts.value = data.counts || {}
  } catch { } finally { loading.value = false }
}

function openCreate() {
  Object.assign(form, { title: '', username: '', priority: 'medium', description: '' })
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    await ticketsAPI.create(form)
    toast.success('Ticket created')
    showModal.value = false
    load()
  } catch (err) { toast.error(err.response?.data?.error || 'Failed') }
  finally { saving.value = false }
}

async function updateStatus(ticket, newStatus) {
  try {
    await ticketsAPI.update(ticket.id, { status: newStatus })
    ticket.status = newStatus
    load()
  } catch { toast.error('Update failed') }
}

async function deleteTicket(t) {
  if (!confirm(`Delete ticket #${t.id}?`)) return
  try { await ticketsAPI.delete(t.id); toast.success('Deleted'); load() }
  catch { toast.error('Failed') }
}

onMounted(load)
</script>
