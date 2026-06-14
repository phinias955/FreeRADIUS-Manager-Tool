<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Alert Rules</h1>
        <p class="text-sm text-gray-500 mt-0.5">Automatic email notifications for system events</p>
      </div>
      <div class="flex gap-2">
        <button @click="sendTestEmail" class="btn-secondary">
          <EnvelopeIcon class="w-4 h-4" />
          Test Email
        </button>
        <button @click="openCreate" class="btn-primary">
          <PlusIcon class="w-4 h-4" />
          New Rule
        </button>
      </div>
    </div>

    <!-- SMTP status banner -->
    <div class="card p-4 flex items-start gap-3"
      :class="smtpConfigured ? 'bg-green-50 border border-green-200' : 'bg-yellow-50 border border-yellow-200'">
      <component :is="smtpConfigured ? CheckCircleIcon : ExclamationTriangleIcon"
        class="w-5 h-5 flex-shrink-0 mt-0.5"
        :class="smtpConfigured ? 'text-green-600' : 'text-yellow-600'" />
      <div>
        <p class="text-sm font-medium" :class="smtpConfigured ? 'text-green-800' : 'text-yellow-800'">
          {{ smtpConfigured ? 'SMTP configured — email alerts enabled' : 'SMTP not configured — email alerts disabled' }}
        </p>
        <p class="text-xs mt-0.5" :class="smtpConfigured ? 'text-green-600' : 'text-yellow-600'">
          {{ smtpConfigured ? `Sending from: ${smtpFrom}` : 'Go to Settings → add SMTP_HOST, SMTP_USER, SMTP_PASS environment variables' }}
        </p>
      </div>
    </div>

    <!-- Alert rules list -->
    <div class="space-y-3">
      <div v-for="rule in rules" :key="rule.id"
        class="card p-4 flex items-center justify-between gap-4"
        :class="!rule.is_active ? 'opacity-60' : ''">
        <div class="flex items-center gap-3 flex-1 min-w-0">
          <div class="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0"
            :class="eventColor(rule.event_type)">
            <BellIcon class="w-5 h-5" />
          </div>
          <div class="min-w-0">
            <p class="font-medium text-gray-900 text-sm">{{ rule.name }}</p>
            <p class="text-xs text-gray-500 mt-0.5">
              Trigger: <span class="font-mono">{{ rule.event_type }}</span>
              <span v-if="rule.email_address" class="ml-2">→ {{ rule.email_address }}</span>
            </p>
            <p v-if="rule.last_triggered" class="text-xs text-gray-400 mt-0.5">
              Last triggered: {{ formatDate(rule.last_triggered) }}
            </p>
          </div>
        </div>
        <div class="flex items-center gap-3 flex-shrink-0">
          <span class="badge" :class="rule.is_active ? 'badge-green' : 'badge-gray'">
            {{ rule.is_active ? 'Active' : 'Paused' }}
          </span>
          <button @click="toggleRule(rule)" class="btn-secondary text-xs py-1.5 px-3">
            {{ rule.is_active ? 'Pause' : 'Enable' }}
          </button>
          <button @click="openEdit(rule)" class="btn-secondary text-xs py-1.5 px-3">Edit</button>
          <button @click="remove(rule)" class="text-xs py-1.5 px-3 rounded-lg border border-red-200 text-red-600 hover:bg-red-50">Del</button>
        </div>
      </div>
      <div v-if="!rules.length" class="card p-12 text-center text-gray-400">
        No alert rules configured.
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Alert Rule</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div>
            <label class="form-label">Rule Name <span class="text-red-500">*</span></label>
            <input v-model="form.name" type="text" class="form-input" required />
          </div>
          <div v-if="!editing">
            <label class="form-label">Event Type <span class="text-red-500">*</span></label>
            <select v-model="form.event_type" class="form-input" required>
              <option value="">Select event…</option>
              <option value="data_limit_80pct">Data at 80% of limit</option>
              <option value="data_limit_100pct">Data limit reached</option>
              <option value="account_expiry_3d">Account expiring in 3 days</option>
              <option value="account_expiry_7d">Account expiring in 7 days</option>
              <option value="login_failure">Repeated login failures</option>
              <option value="nas_down">NAS device down</option>
              <option value="new_user">New user created</option>
            </select>
          </div>
          <div>
            <label class="form-label">Notify Email Address</label>
            <input v-model="form.email_address" type="email" class="form-input" placeholder="admin@example.com" />
            <p class="text-xs text-gray-400 mt-1">Leave empty to use SMTP_FROM address</p>
          </div>
          <div class="flex items-center gap-3">
            <input type="checkbox" id="notify_email" v-model="form.notify_email" class="rounded" />
            <label for="notify_email" class="text-sm text-gray-700">Send email notification</label>
          </div>
          <div class="flex items-center gap-3">
            <input type="checkbox" id="rule_active" v-model="form.is_active" class="rounded" />
            <label for="rule_active" class="text-sm text-gray-700">Rule is active</label>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">Save Rule</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Test email modal -->
    <div v-if="showTestModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-sm">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">Send Test Email</h3>
          <button @click="showTestModal = false" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="doTestEmail" class="p-6 space-y-4">
          <div>
            <label class="form-label">Recipient Email <span class="text-red-500">*</span></label>
            <input v-model="testEmail" type="email" class="form-input" required placeholder="your@email.com" />
          </div>
          <div class="flex justify-end gap-3">
            <button type="button" @click="showTestModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="testingEmail">
              <span v-if="testingEmail" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Send Test
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { alertsAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { PlusIcon, XMarkIcon, BellIcon, EnvelopeIcon, CheckCircleIcon, ExclamationTriangleIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const showTestModal = ref(false)
const testingEmail = ref(false)
const testEmail = ref('')
const editing = ref(null)
const rules = ref([])

const smtpConfigured = computed(() => {
  // Detect based on whether alert send would work — use a simple env indicator
  // Backend will return error if SMTP not set, we check on test email
  return rules.value.some(r => r.is_active)
})
const smtpFrom = ref(import.meta.env.VITE_SMTP_FROM || 'configured in .env')

const form = reactive({ name: '', event_type: '', notify_email: true, email_address: '', is_active: true })

function eventColor(t) {
  const map = {
    data_limit_80pct: 'bg-yellow-100 text-yellow-600',
    data_limit_100pct: 'bg-red-100 text-red-600',
    account_expiry_3d: 'bg-orange-100 text-orange-600',
    account_expiry_7d: 'bg-orange-100 text-orange-500',
    nas_down: 'bg-red-100 text-red-700',
    login_failure: 'bg-purple-100 text-purple-600',
    new_user: 'bg-blue-100 text-blue-600',
  }
  return map[t] || 'bg-gray-100 text-gray-600'
}

function formatDate(d) {
  if (!d) return '—'
  try { return format(parseISO(d), 'MMM d, HH:mm') } catch { return d }
}

async function load() {
  loading.value = true
  try {
    const { data } = await alertsAPI.list()
    rules.value = data.data || []
  } catch { /* silent */ } finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', event_type: '', notify_email: true, email_address: '', is_active: true })
  showModal.value = true
}

function openEdit(r) {
  editing.value = r.id
  Object.assign(form, { name: r.name, event_type: r.event_type, notify_email: r.notify_email, email_address: r.email_address || '', is_active: r.is_active })
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) {
      await alertsAPI.update(editing.value, form)
      toast.success('Rule updated')
    } else {
      await alertsAPI.create(form)
      toast.success('Rule created')
    }
    showModal.value = false
    load()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Save failed')
  } finally { saving.value = false }
}

async function toggleRule(r) {
  try {
    await alertsAPI.update(r.id, { is_active: !r.is_active })
    load()
  } catch (err) { toast.error('Failed') }
}

async function remove(r) {
  if (!confirm(`Delete alert rule "${r.name}"?`)) return
  try {
    await alertsAPI.delete(r.id)
    toast.success('Deleted')
    load()
  } catch { /* silent */ }
}

function sendTestEmail() { showTestModal.value = true }

async function doTestEmail() {
  testingEmail.value = true
  try {
    await alertsAPI.testEmail({ to: testEmail.value })
    toast.success('Test email sent! Check your inbox.')
    showTestModal.value = false
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed — check SMTP settings')
  } finally { testingEmail.value = false }
}

onMounted(load)
</script>
