<template>
  <div class="space-y-5">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">RADIUS Users</h1>
        <p class="text-sm text-gray-500 mt-0.5">Manage network access users</p>
      </div>
      <div class="flex items-center gap-2" v-if="authStore.isAdmin">
        <label class="btn-secondary cursor-pointer">
          <ArrowUpTrayIcon class="w-4 h-4" />
          Import CSV
          <input type="file" accept=".csv" class="hidden" @change="handleImport" />
        </label>
        <button @click="exportUsers" class="btn-secondary">
          <ArrowDownTrayIcon class="w-4 h-4" />
          Export
        </button>
        <button @click="showImport = true" class="btn-secondary">
          <ArrowUpTrayIcon class="w-4 h-4" />
          Import CSV
        </button>
        <button @click="openCreate" class="btn-primary">
          <PlusIcon class="w-4 h-4" />
          Add User
        </button>
      </div>
    </div>

    <!-- Filters -->
    <div class="card p-4">
      <div class="flex flex-wrap gap-3">
        <div class="flex-1 min-w-[200px]">
          <input
            v-model="filters.search"
            type="text"
            class="form-input"
            placeholder="Search by username, email, name..."
            @input="debouncedSearch"
          />
        </div>
        <select v-model="filters.status" class="form-select w-40" @change="loadUsers">
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="suspended">Suspended</option>
          <option value="expired">Expired</option>
        </select>
        <input
          v-model="filters.department"
          type="text"
          class="form-input w-40"
          placeholder="Department..."
          @input="debouncedSearch"
        />
      </div>
    </div>

    <!-- Table -->
    <div class="card p-0 overflow-hidden">
      <div v-if="loading" class="h-64 flex items-center justify-center">
        <span class="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full spinner"></span>
      </div>

      <div v-else>
        <div class="table-container rounded-none border-0">
          <table class="table">
            <thead>
              <tr>
                <th>Username</th>
                <th>Name / Email</th>
                <th>Department</th>
                <th>Status</th>
                <th>Devices</th>
                <th>Sessions</th>
                <th>Expires</th>
                <th class="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!users.length">
                <td colspan="8" class="text-center text-gray-400 py-12">No users found</td>
              </tr>
              <tr v-for="user in users" :key="user.id">
                <td>
                  <span class="font-mono font-medium text-gray-900">{{ user.username }}</span>
                </td>
                <td>
                  <div class="text-sm font-medium text-gray-900">{{ user.full_name || '—' }}</div>
                  <div class="text-xs text-gray-500">{{ user.email || '—' }}</div>
                </td>
                <td class="text-gray-500">{{ user.department || '—' }}</td>
                <td>
                  <span :class="statusBadge(user.status)" class="badge">
                    {{ user.status }}
                  </span>
                </td>
                <td>
                  <div class="flex items-center gap-1.5">
                    <span class="text-sm font-medium">{{ user.active_sessions }}/{{ user.device_limit }}</span>
                    <div class="w-16 bg-gray-100 rounded-full h-1.5">
                      <div
                        class="h-1.5 rounded-full"
                        :class="user.active_sessions >= user.device_limit ? 'bg-red-500' : 'bg-green-500'"
                        :style="{ width: `${(user.active_sessions / user.device_limit) * 100}%` }"
                      ></div>
                    </div>
                  </div>
                </td>
                <td>
                  <span class="text-sm" :class="user.active_sessions > 0 ? 'text-green-600 font-medium' : 'text-gray-400'">
                    {{ user.active_sessions }} active
                  </span>
                </td>
                <td class="text-gray-500 text-xs">
                  {{ user.account_expiry ? formatDate(user.account_expiry) : 'Never' }}
                </td>
                <td>
                  <div class="flex items-center justify-end gap-1">
                    <button
                      @click="openResetPassword(user)"
                      class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded"
                      title="Reset Password"
                    >
                      <KeyIcon class="w-4 h-4" />
                    </button>
                    <template v-if="authStore.isAdmin">
                      <button
                        @click="openEdit(user)"
                        class="p-1.5 text-gray-400 hover:text-gray-700 hover:bg-gray-100 rounded"
                        title="Edit"
                      >
                        <PencilIcon class="w-4 h-4" />
                      </button>
                      <button
                        v-if="user.status === 'active'"
                        @click="toggleStatus(user)"
                        class="p-1.5 text-gray-400 hover:text-yellow-600 hover:bg-yellow-50 rounded"
                        title="Suspend"
                      >
                        <PauseIcon class="w-4 h-4" />
                      </button>
                      <button
                        v-else
                        @click="toggleStatus(user)"
                        class="p-1.5 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded"
                        title="Activate"
                      >
                        <PlayIcon class="w-4 h-4" />
                      </button>
                      <button
                        v-if="user.active_sessions > 0"
                        @click="disconnectUser(user)"
                        class="p-1.5 text-gray-400 hover:text-orange-600 hover:bg-orange-50 rounded"
                        title="Disconnect Sessions"
                      >
                        <SignalSlashIcon class="w-4 h-4" />
                      </button>
                      <button
                        @click="confirmDelete(user)"
                        class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
                        title="Delete"
                      >
                        <TrashIcon class="w-4 h-4" />
                      </button>
                    </template>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div class="flex items-center justify-between px-4 py-3 border-t border-gray-200">
          <p class="text-sm text-gray-500">
            Showing {{ offset + 1 }}–{{ Math.min(offset + limit, total) }} of {{ total }} users
          </p>
          <div class="flex gap-2">
            <button @click="prevPage" :disabled="page === 1" class="btn-secondary py-1.5 px-3">
              Previous
            </button>
            <button @click="nextPage" :disabled="offset + limit >= total" class="btn-secondary py-1.5 px-3">
              Next
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <UserModal
      v-if="showModal"
      :user="editingUser"
      @close="showModal = false"
      @saved="onSaved"
    />

    <!-- Reset Password Modal -->
    <ResetPasswordModal
      v-if="showResetModal"
      :user="editingUser"
      @close="showResetModal = false"
      @saved="onPasswordReset"
    />

    <!-- Delete confirm -->
    <ConfirmDialog
      v-if="showDeleteConfirm"
      title="Delete User"
      :message="`Are you sure you want to delete '${editingUser?.username}'? This action cannot be undone.`"
      confirm-label="Delete"
      confirm-class="btn-danger"
      @confirm="deleteUser"
      @cancel="showDeleteConfirm = false"
    />

    <!-- Bulk Import CSV Modal -->
    <div v-if="showImport" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-lg">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">Bulk Import Users (CSV)</h3>
          <button @click="closeImport" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <div v-if="!importResult" class="p-6 space-y-4">
          <div class="bg-blue-50 border border-blue-200 rounded-xl p-3 text-xs text-blue-800">
            <p class="font-semibold mb-1">Required CSV columns:</p>
            <code class="font-mono">username, password</code>
            <p class="mt-1">Optional: <code class="font-mono">email, plan_name, data_limit_mb, validity_days, description</code></p>
          </div>
          <div
            class="border-2 border-dashed border-gray-300 rounded-xl p-8 text-center cursor-pointer hover:border-blue-400 transition-colors"
            @click="$refs.csvInput.click()"
            @dragover.prevent
            @drop.prevent="onDrop">
            <ArrowUpTrayIcon class="w-10 h-10 text-gray-300 mx-auto mb-3" />
            <p class="text-sm text-gray-600">{{ csvFile ? csvFile.name : 'Click to select or drag & drop CSV' }}</p>
            <p class="text-xs text-gray-400 mt-1">Max recommended: 5000 rows</p>
            <input ref="csvInput" type="file" accept=".csv,text/csv" class="hidden" @change="onFileSelect" />
          </div>
          <div class="flex justify-end gap-3">
            <button @click="closeImport" class="btn-secondary">Cancel</button>
            <button @click="doImport" :disabled="!csvFile || importing" class="btn-primary">
              <span v-if="importing" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Import
            </button>
          </div>
        </div>
        <div v-else class="p-6 space-y-4">
          <div class="grid grid-cols-3 gap-3 text-center">
            <div class="bg-green-50 rounded-xl p-3">
              <p class="text-2xl font-bold text-green-600">{{ importResult.created }}</p>
              <p class="text-xs text-green-700">Created</p>
            </div>
            <div class="bg-yellow-50 rounded-xl p-3">
              <p class="text-2xl font-bold text-yellow-600">{{ importResult.skipped }}</p>
              <p class="text-xs text-yellow-700">Skipped</p>
            </div>
            <div class="bg-red-50 rounded-xl p-3">
              <p class="text-2xl font-bold text-red-600">{{ importResult.failed }}</p>
              <p class="text-xs text-red-700">Failed</p>
            </div>
          </div>
          <div v-if="importResult.results?.length" class="max-h-48 overflow-y-auto">
            <table class="w-full text-xs">
              <thead><tr class="text-gray-500"><th class="text-left py-1">Row</th><th class="text-left">Username</th><th class="text-left">Status</th><th class="text-left">Note</th></tr></thead>
              <tbody>
                <tr v-for="r in importResult.results.filter(x => x.status !== 'created')" :key="r.row" class="border-t border-gray-100">
                  <td class="py-1">{{ r.row }}</td>
                  <td>{{ r.username || '—' }}</td>
                  <td><span class="badge text-xs" :class="r.status === 'skip' ? 'badge-yellow' : 'badge-red'">{{ r.status }}</span></td>
                  <td class="text-gray-400">{{ r.message }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="flex justify-end">
            <button @click="closeImport" class="btn-primary">Done</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/store/auth'
import { useToast } from 'vue-toastification'
import { radiusUsersAPI, importAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import {
  PlusIcon, PencilIcon, TrashIcon, KeyIcon, PauseIcon, PlayIcon,
  ArrowUpTrayIcon, ArrowDownTrayIcon, SignalSlashIcon, XMarkIcon,
} from '@heroicons/vue/24/outline'
import UserModal from '@/components/users/UserModal.vue'
import ResetPasswordModal from '@/components/users/ResetPasswordModal.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const authStore = useAuthStore()
const toast = useToast()

const users = ref([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const limit = 20
const offset = computed(() => (page.value - 1) * limit)

const filters = ref({ search: '', status: '', department: '' })
const showModal = ref(false)
const showResetModal = ref(false)
const showDeleteConfirm = ref(false)
const editingUser = ref(null)

let searchTimer

function debouncedSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => loadUsers(), 400)
}

async function loadUsers() {
  loading.value = true
  try {
    const { data } = await radiusUsersAPI.list({
      page: page.value,
      limit,
      search: filters.value.search,
      status: filters.value.status,
      department: filters.value.department,
    })
    users.value = data.data || []
    total.value = data.total || 0
  } catch {
    toast.error('Failed to load users')
  } finally {
    loading.value = false
  }
}

function nextPage() {
  if (offset.value + limit < total.value) {
    page.value++
    loadUsers()
  }
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    loadUsers()
  }
}

function openCreate() {
  editingUser.value = null
  showModal.value = true
}

function openEdit(user) {
  editingUser.value = { ...user }
  showModal.value = true
}

function openResetPassword(user) {
  editingUser.value = user
  showResetModal.value = true
}

function confirmDelete(user) {
  editingUser.value = user
  showDeleteConfirm.value = true
}

async function deleteUser() {
  showDeleteConfirm.value = false
  try {
    await radiusUsersAPI.delete(editingUser.value.id)
    toast.success('User deleted successfully')
    loadUsers()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed to delete user')
  }
}

async function toggleStatus(user) {
  try {
    if (user.status === 'active') {
      await radiusUsersAPI.suspend(user.id)
      toast.success(`${user.username} suspended`)
    } else {
      await radiusUsersAPI.activate(user.id)
      toast.success(`${user.username} activated`)
    }
    loadUsers()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed to change status')
  }
}

async function disconnectUser(user) {
  try {
    const { data } = await radiusUsersAPI.disconnect(user.id)
    toast.success(data.message)
    loadUsers()
  } catch {
    toast.error('Failed to disconnect user')
  }
}

async function handleImport(event) {
  const file = event.target.files[0]
  if (!file) return
  const formData = new FormData()
  formData.append('file', file)
  try {
    const { data } = await radiusUsersAPI.import(formData)
    toast.success(`Imported ${data.created} users`)
    if (data.errors?.length) {
      console.warn('Import errors:', data.errors)
    }
    loadUsers()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Import failed')
  }
  event.target.value = ''
}

async function exportUsers() {
  try {
    const { data } = await radiusUsersAPI.export()
    const url = URL.createObjectURL(new Blob([data]))
    const a = document.createElement('a')
    a.href = url
    a.download = 'radius_users.csv'
    a.click()
    URL.revokeObjectURL(url)
  } catch {
    toast.error('Export failed')
  }
}

function onSaved() {
  showModal.value = false
  loadUsers()
}

function onPasswordReset() {
  showResetModal.value = false
  toast.success('Password reset successfully')
}

function statusBadge(status) {
  return { active: 'badge-green', suspended: 'badge-red', expired: 'badge-yellow' }[status] || 'badge-gray'
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  try { return format(parseISO(dateStr), 'MMM d, yyyy') } catch { return dateStr }
}

// ── Bulk Import ──────────────────────────────────────────────────────────────
const showImport = ref(false)
const csvFile = ref(null)
const importing = ref(false)
const importResult = ref(null)

function onFileSelect(e) { csvFile.value = e.target.files[0] }
function onDrop(e) { csvFile.value = e.dataTransfer.files[0] }
function closeImport() { showImport.value = false; csvFile.value = null; importResult.value = null; loadUsers() }

async function doImport() {
  if (!csvFile.value) return
  importing.value = true
  try {
    const formData = new FormData()
    formData.append('file', csvFile.value)
    const { data } = await importAPI.importCSV(formData)
    importResult.value = data
    toast.success(data.message)
  } catch (err) {
    toast.error(err.response?.data?.error || 'Import failed')
  } finally { importing.value = false }
}

onMounted(loadUsers)
</script>
