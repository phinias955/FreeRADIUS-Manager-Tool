<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">API Keys</h1>
        <p class="text-sm text-gray-500 mt-0.5">External integration keys for automated access to the RADIUS Manager API</p>
      </div>
      <button @click="showCreate = true" class="btn-primary">
        <PlusIcon class="w-4 h-4" />
        Generate Key
      </button>
    </div>

    <!-- Docs card -->
    <div class="card bg-blue-50 border border-blue-200 p-4 flex items-start gap-3">
      <InformationCircleIcon class="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
      <div class="text-sm text-blue-800">
        <p class="font-medium">API Key Authentication</p>
        <p class="mt-0.5 text-blue-700">Include the key in every request header:</p>
        <code class="block mt-1 bg-blue-100 rounded px-2 py-1 text-xs font-mono">Authorization: ApiKey rmk_your_key_here</code>
        <p class="mt-1 text-xs text-blue-600">Keys are shown only once at creation — store them securely.</p>
      </div>
    </div>

    <!-- Keys table -->
    <div class="card p-0 overflow-hidden">
      <table class="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Prefix</th>
            <th>Permissions</th>
            <th>Created By</th>
            <th>Last Used</th>
            <th>Expires</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="8" class="text-center py-12 text-gray-400">Loading…</td>
          </tr>
          <tr v-else-if="!keys.length">
            <td colspan="8" class="text-center py-12 text-gray-400">No API keys — generate one to allow external integrations</td>
          </tr>
          <tr v-for="key in keys" :key="key.id" :class="!key.is_active ? 'opacity-50' : ''">
            <td class="font-medium">{{ key.name }}</td>
            <td><code class="text-xs bg-gray-100 px-2 py-0.5 rounded font-mono">{{ key.key_prefix }}…</code></td>
            <td>
              <div class="flex flex-wrap gap-1">
                <span v-for="p in key.permissions" :key="p" class="badge badge-blue text-xs">{{ p }}</span>
              </div>
            </td>
            <td class="text-gray-500 text-sm">{{ key.created_by || '—' }}</td>
            <td class="text-xs text-gray-500">{{ key.last_used ? formatDate(key.last_used) : 'Never' }}</td>
            <td class="text-xs text-gray-500">{{ key.expires_at ? formatDate(key.expires_at) : '∞ Never' }}</td>
            <td>
              <span class="badge" :class="key.is_active ? 'badge-green' : 'badge-red'">
                {{ key.is_active ? 'Active' : 'Revoked' }}
              </span>
            </td>
            <td>
              <div class="flex gap-1">
                <button v-if="key.is_active" @click="revoke(key)" class="text-xs px-2 py-1 rounded bg-yellow-100 text-yellow-700 hover:bg-yellow-200">Revoke</button>
                <button @click="del(key)" class="text-xs px-2 py-1 rounded bg-red-100 text-red-600 hover:bg-red-200">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreate" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">Generate API Key</h3>
          <button @click="showCreate = false; newKey = null" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>

        <!-- Show new key (after creation) -->
        <div v-if="newKey" class="p-6 space-y-4">
          <div class="bg-green-50 border border-green-200 rounded-xl p-4">
            <p class="font-semibold text-green-800 mb-2">Key created successfully!</p>
            <p class="text-xs text-green-600 mb-2">Copy this key now — it will NOT be shown again:</p>
            <div class="flex items-center gap-2">
              <code class="flex-1 text-xs bg-white border border-green-300 rounded px-3 py-2 font-mono break-all">{{ newKey }}</code>
              <button @click="copyKey" class="btn-secondary text-xs py-2 px-3">
                <ClipboardDocumentIcon class="w-4 h-4" />
              </button>
            </div>
          </div>
          <div class="flex justify-end">
            <button @click="showCreate = false; newKey = null; load()" class="btn-primary">Done</button>
          </div>
        </div>

        <!-- Create form -->
        <form v-else @submit.prevent="generate" class="p-6 space-y-4">
          <div>
            <label class="form-label">Key Name <span class="text-red-500">*</span></label>
            <input v-model="createForm.name" type="text" class="form-input" required placeholder="e.g. Monitoring System" />
          </div>
          <div>
            <label class="form-label">Permissions</label>
            <div class="flex flex-wrap gap-3 mt-1">
              <label v-for="perm in availablePerms" :key="perm" class="flex items-center gap-2 text-sm">
                <input type="checkbox" v-model="createForm.permissions" :value="perm" class="rounded" />
                {{ perm }}
              </label>
            </div>
          </div>
          <div>
            <label class="form-label">Expires At (optional)</label>
            <input v-model="createForm.expires_at" type="date" class="form-input" />
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showCreate = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Generate
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
import { apiKeysAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { PlusIcon, XMarkIcon, InformationCircleIcon, ClipboardDocumentIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const loading = ref(false)
const saving = ref(false)
const showCreate = ref(false)
const newKey = ref(null)
const keys = ref([])
const availablePerms = ['read', 'write', 'admin']

const createForm = reactive({ name: '', permissions: ['read'], expires_at: '' })

function formatDate(d) {
  try { return format(parseISO(d), 'MMM d, yyyy') } catch { return d }
}

async function load() {
  loading.value = true
  try {
    const { data } = await apiKeysAPI.list()
    keys.value = data.data || []
  } catch { /* silent */ } finally { loading.value = false }
}

async function generate() {
  saving.value = true
  try {
    const payload = { ...createForm }
    if (!payload.expires_at) delete payload.expires_at
    const { data } = await apiKeysAPI.create(payload)
    newKey.value = data.key
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed')
  } finally { saving.value = false }
}

function copyKey() {
  navigator.clipboard.writeText(newKey.value)
  toast.success('Key copied to clipboard')
}

async function revoke(key) {
  if (!confirm(`Revoke key "${key.name}"? This cannot be undone.`)) return
  try {
    await apiKeysAPI.revoke(key.id)
    toast.success('Key revoked')
    load()
  } catch { toast.error('Failed') }
}

async function del(key) {
  if (!confirm(`Permanently delete key "${key.name}"?`)) return
  try {
    await apiKeysAPI.delete(key.id)
    toast.success('Key deleted')
    load()
  } catch { toast.error('Failed') }
}

onMounted(load)
</script>
