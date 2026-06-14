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
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/store/auth'
import { useToast } from 'vue-toastification'
import { radiusUsersAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import {
  PlusIcon, PencilIcon, TrashIcon, KeyIcon, PauseIcon, PlayIcon,
  ArrowUpTrayIcon, ArrowDownTrayIcon, SignalSlashIcon,
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

onMounted(loadUsers)
</script>
