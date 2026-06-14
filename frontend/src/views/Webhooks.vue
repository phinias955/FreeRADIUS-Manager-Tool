<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Webhooks</h1>
        <p class="text-sm text-gray-500 mt-0.5">Send real-time event notifications to external systems</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isSuperAdmin">
        <PlusIcon class="w-4 h-4" />
        Add Webhook
      </button>
    </div>

    <!-- Info banner -->
    <div class="card bg-blue-50 border border-blue-200 p-4 flex gap-3">
      <InformationCircleIcon class="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5" />
      <div class="text-sm text-blue-700">
        <p class="font-medium">How webhooks work</p>
        <p class="mt-0.5 text-blue-600 text-xs">RADIUS Manager sends a signed JSON POST to your URL when events occur. Verify the <code class="bg-blue-100 px-1 rounded">X-RADIUS-Signature</code> header using your secret. Leave events empty to receive all events.</p>
      </div>
    </div>

    <!-- Event list -->
    <div class="flex flex-wrap gap-1.5 text-xs">
      <span class="text-gray-500">Available events:</span>
      <span v-for="e in availableEvents" :key="e" class="px-2 py-0.5 bg-gray-100 text-gray-700 rounded-md font-mono">{{ e }}</span>
    </div>

    <!-- Webhooks list -->
    <div v-if="!hooks.length && !loading" class="card text-center py-16 text-gray-400">
      <LinkIcon class="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p class="font-medium">No webhooks configured</p>
      <p class="text-sm mt-1">Add a webhook to integrate with Zapier, Slack, or your own system</p>
    </div>

    <div class="space-y-3">
      <div v-for="hook in hooks" :key="hook.id" class="card">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-xl flex items-center justify-center"
              :class="hook.is_active ? 'bg-blue-100' : 'bg-gray-100'">
              <LinkIcon class="w-5 h-5" :class="hook.is_active ? 'text-blue-600' : 'text-gray-400'" />
            </div>
            <div>
              <p class="font-semibold text-gray-900">{{ hook.name }}</p>
              <p class="font-mono text-xs text-gray-400 mt-0.5 truncate max-w-xs">{{ hook.url }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <span class="badge" :class="hook.is_active ? 'badge-green' : 'badge-gray'">
              {{ hook.is_active ? 'Active' : 'Off' }}
            </span>
            <span v-if="hook.fail_count > 0" class="badge badge-red">{{ hook.fail_count }} fails</span>
          </div>
        </div>

        <div class="mt-3 flex flex-wrap gap-1">
          <span v-if="!hook.events?.length" class="text-xs bg-purple-100 text-purple-700 px-2 py-0.5 rounded-md">all events</span>
          <span v-for="e in hook.events" :key="e" class="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-md font-mono">{{ e }}</span>
        </div>

        <div class="flex items-center gap-2 mt-3 pt-3 border-t border-gray-100">
          <span v-if="hook.last_triggered" class="text-xs text-gray-400">Last: {{ formatDate(hook.last_triggered) }}</span>
          <span v-else class="text-xs text-gray-400">Never triggered</span>
          <div class="ml-auto flex gap-2">
            <button @click="viewLogs(hook)" class="btn-secondary text-xs py-1.5">Logs</button>
            <button @click="testHook(hook)" class="btn-secondary text-xs py-1.5" :disabled="testing === hook.id">
              <span v-if="testing === hook.id" class="w-3.5 h-3.5 border-2 border-gray-400 border-t-gray-700 rounded-full spinner"></span>
              <BoltIcon v-else class="w-3.5 h-3.5" />
              Test
            </button>
            <button @click="openEdit(hook)" class="btn-secondary text-xs py-1.5">Edit</button>
            <button @click="removeHook(hook)" class="p-1.5 rounded-lg border border-red-200 text-red-500 hover:bg-red-50">
              <TrashIcon class="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        <!-- Logs panel -->
        <div v-if="logsFor === hook.id" class="mt-3 pt-3 border-t border-gray-100">
          <div class="flex items-center justify-between mb-2">
            <h4 class="text-xs font-medium text-gray-600">Recent Deliveries</h4>
            <button @click="logsFor = null" class="text-xs text-gray-400 hover:text-gray-600">Hide</button>
          </div>
          <div v-if="!logs.length" class="text-xs text-gray-400 py-2">No delivery history</div>
          <div v-for="l in logs" :key="l.id"
            class="flex items-center justify-between py-1.5 border-b border-gray-50 last:border-0">
            <div class="flex items-center gap-2">
              <span class="w-2 h-2 rounded-full" :class="l.success ? 'bg-green-500' : 'bg-red-500'"></span>
              <span class="font-mono text-xs text-gray-600">{{ l.event }}</span>
            </div>
            <div class="flex items-center gap-3 text-xs text-gray-400">
              <span>HTTP {{ l.status_code }}</span>
              <span>{{ l.created_at?.slice(0,16) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Webhook</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div>
            <label class="form-label">Name <span class="text-red-500">*</span></label>
            <input v-model="form.name" type="text" class="form-input" required placeholder="Slack Integration" />
          </div>
          <div>
            <label class="form-label">URL <span class="text-red-500">*</span></label>
            <input v-model="form.url" type="url" class="form-input" required placeholder="https://hooks.slack.com/..." />
          </div>
          <div>
            <label class="form-label">Secret (for HMAC signature)</label>
            <input v-model="form.secret" type="text" class="form-input font-mono text-sm" placeholder="Optional signing secret" />
          </div>
          <div>
            <label class="form-label">Events (blank = all events)</label>
            <div class="flex flex-wrap gap-1.5 p-3 border border-gray-200 rounded-xl">
              <button v-for="e in availableEvents" :key="e" type="button"
                @click="toggleEvent(e)"
                class="text-xs px-2.5 py-1 rounded-lg border transition-colors"
                :class="form.events.includes(e) ? 'bg-blue-600 border-blue-600 text-white' : 'border-gray-200 text-gray-600 hover:bg-gray-50'">
                {{ e }}
              </button>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="hook_active" v-model="form.is_active" class="rounded" />
            <label for="hook_active" class="text-sm text-gray-700">Webhook is active</label>
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
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { useAuthStore } from '@/store/auth'
import { webhooksAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { PlusIcon, XMarkIcon, TrashIcon, LinkIcon, BoltIcon, InformationCircleIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const testing = ref(null)
const showModal = ref(false)
const editing = ref(null)
const logsFor = ref(null)
const hooks = ref([])
const logs = ref([])
const form = reactive({ name: '', url: '', secret: '', events: [], is_active: true })

const availableEvents = ['user.created', 'user.suspended', 'user.expired', 'session.started', 'session.stopped', 'invoice.paid', 'ticket.created', 'nas.down', 'test']

function formatDate(d) {
  try { return format(parseISO(d), 'MMM d HH:mm') } catch { return d }
}

function toggleEvent(e) {
  const idx = form.events.indexOf(e)
  if (idx >= 0) form.events.splice(idx, 1)
  else form.events.push(e)
}

async function load() {
  loading.value = true
  try { const { data } = await webhooksAPI.list(); hooks.value = data.data || [] }
  catch { } finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', url: '', secret: '', events: [], is_active: true })
  showModal.value = true
}

function openEdit(hook) {
  editing.value = hook.id
  Object.assign(form, { name: hook.name, url: hook.url, secret: '', events: [...(hook.events || [])], is_active: hook.is_active })
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) { await webhooksAPI.update(editing.value, form); toast.success('Updated') }
    else { await webhooksAPI.create(form); toast.success('Created') }
    showModal.value = false; load()
  } catch (err) { toast.error(err.response?.data?.error || 'Failed') }
  finally { saving.value = false }
}

async function testHook(hook) {
  testing.value = hook.id
  try {
    const { data } = await webhooksAPI.test(hook.id)
    toast.success(`Test delivered — HTTP ${data.status_code}`)
    load()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Delivery failed')
  } finally { testing.value = null }
}

async function viewLogs(hook) {
  if (logsFor.value === hook.id) { logsFor.value = null; return }
  try {
    const { data } = await webhooksAPI.logs(hook.id)
    logs.value = data.data || []
    logsFor.value = hook.id
  } catch { toast.error('Failed to load logs') }
}

async function removeHook(hook) {
  if (!confirm(`Delete webhook "${hook.name}"?`)) return
  try { await webhooksAPI.delete(hook.id); toast.success('Deleted'); load() }
  catch { toast.error('Failed') }
}

onMounted(load)
</script>
