<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Admin Users</h1>
        <p class="text-sm text-gray-500 mt-0.5">Manage system administrators and operators</p>
      </div>
      <button @click="openCreate" class="btn-primary">
        <PlusIcon class="w-4 h-4" />
        Add Admin
      </button>
    </div>

    <div class="card p-0 overflow-hidden">
      <div v-if="loading" class="h-48 flex items-center justify-center">
        <span class="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full spinner"></span>
      </div>

      <div v-else class="table-container rounded-none border-0">
        <table class="table">
          <thead>
            <tr>
              <th>Username</th>
              <th>Name / Email</th>
              <th>Role</th>
              <th>MFA</th>
              <th>Status</th>
              <th>Last Login</th>
              <th class="text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!users.length">
              <td colspan="7" class="text-center text-gray-400 py-12">No admin users found</td>
            </tr>
            <tr v-for="u in users" :key="u.id">
              <td class="font-mono font-medium">{{ u.username }}</td>
              <td>
                <div class="text-sm font-medium">{{ u.full_name || '—' }}</div>
                <div class="text-xs text-gray-500">{{ u.email }}</div>
              </td>
              <td>
                <span :class="roleBadge(u.role)" class="badge capitalize">
                  {{ u.role.replace('_', ' ') }}
                </span>
              </td>
              <td>
                <span :class="u.mfa_enabled ? 'badge-green' : 'badge-gray'" class="badge">
                  {{ u.mfa_enabled ? 'Enabled' : 'Disabled' }}
                </span>
              </td>
              <td>
                <span :class="u.is_active ? 'badge-green' : 'badge-red'" class="badge">
                  {{ u.is_active ? 'Active' : 'Disabled' }}
                </span>
              </td>
              <td class="text-xs text-gray-500">{{ u.last_login ? formatDate(u.last_login) : 'Never' }}</td>
              <td>
                <div class="flex justify-end gap-1">
                  <button
                    v-if="u.id !== currentUserId"
                    @click="openEdit(u)"
                    class="p-1.5 text-gray-400 hover:text-gray-700 hover:bg-gray-100 rounded"
                  >
                    <PencilIcon class="w-4 h-4" />
                  </button>
                  <button
                    v-if="u.id !== currentUserId"
                    @click="confirmDelete(u)"
                    class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
                  >
                    <TrashIcon class="w-4 h-4" />
                  </button>
                  <span v-if="u.id === currentUserId" class="text-xs text-gray-400 px-2">You</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <AdminUserModal
      v-if="showModal"
      :user="editingUser"
      @close="showModal = false"
      @saved="onSaved"
    />

    <ConfirmDialog
      v-if="showDeleteConfirm"
      title="Delete Admin User"
      :message="`Delete admin '${editingUser?.username}'? They will lose access immediately.`"
      @confirm="deleteUser"
      @cancel="showDeleteConfirm = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/store/auth'
import { useToast } from 'vue-toastification'
import { adminAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { PlusIcon, PencilIcon, TrashIcon } from '@heroicons/vue/24/outline'
import AdminUserModal from '@/components/users/AdminUserModal.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const authStore = useAuthStore()
const toast = useToast()
const users = ref([])
const loading = ref(false)
const showModal = ref(false)
const showDeleteConfirm = ref(false)
const editingUser = ref(null)

const currentUserId = computed(() => authStore.user?.id)

async function loadUsers() {
  loading.value = true
  try {
    const { data } = await adminAPI.list({ limit: 100 })
    users.value = data.data || []
  } catch { toast.error('Failed to load admin users') }
  finally { loading.value = false }
}

function openCreate() { editingUser.value = null; showModal.value = true }
function openEdit(u) { editingUser.value = { ...u }; showModal.value = true }
function confirmDelete(u) { editingUser.value = u; showDeleteConfirm.value = true }

async function deleteUser() {
  showDeleteConfirm.value = false
  try {
    await adminAPI.delete(editingUser.value.id)
    toast.success('Admin user deleted')
    loadUsers()
  } catch (err) { toast.error(err.response?.data?.error || 'Delete failed') }
}

function onSaved() { showModal.value = false; loadUsers() }

function roleBadge(role) {
  return { super_admin: 'badge-red', admin: 'badge-blue', operator: 'badge-gray' }[role] || 'badge-gray'
}

function formatDate(dateStr) {
  try { return format(parseISO(dateStr), 'MMM d, yyyy HH:mm') } catch { return dateStr }
}

onMounted(loadUsers)
</script>
