<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Organizations</h1>
        <p class="text-sm text-gray-500 mt-0.5">Manage resellers, branches, and tenant groups</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isSuperAdmin">
        <PlusIcon class="w-4 h-4" />
        New Organization
      </button>
    </div>

    <!-- Summary bar -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-blue-600">{{ orgs.length }}</p>
        <p class="text-xs text-gray-500 mt-0.5">Total Orgs</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-green-600">{{ orgs.filter(o=>o.is_active).length }}</p>
        <p class="text-xs text-gray-500 mt-0.5">Active</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-purple-600">{{ totalUsers }}</p>
        <p class="text-xs text-gray-500 mt-0.5">Total Users</p>
      </div>
      <div class="card p-4 text-center">
        <p class="text-2xl font-bold text-orange-500">{{ totalNAS }}</p>
        <p class="text-xs text-gray-500 mt-0.5">NAS Devices</p>
      </div>
    </div>

    <!-- Org cards -->
    <div v-if="loading" class="py-12 text-center text-gray-400">Loading…</div>
    <div v-else-if="!orgs.length" class="card text-center py-16 text-gray-400">
      <BuildingOffice2Icon class="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p class="font-medium">No organizations yet</p>
      <p class="text-sm mt-1">Create organizations to manage resellers and client groups</p>
    </div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-for="org in orgs" :key="org.id"
        class="card hover:shadow-md transition-shadow"
        :class="!org.is_active ? 'opacity-60' : ''">
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl bg-purple-100 flex items-center justify-center flex-shrink-0">
              <BuildingOffice2Icon class="w-5 h-5 text-purple-600" />
            </div>
            <div>
              <h3 class="font-semibold text-gray-900">{{ org.name }}</h3>
              <p class="text-xs text-gray-400 font-mono mt-0.5">{{ org.slug }}</p>
            </div>
          </div>
          <span class="badge" :class="org.is_active ? 'badge-green' : 'badge-gray'">
            {{ org.is_active ? 'Active' : 'Inactive' }}
          </span>
        </div>

        <div class="grid grid-cols-2 gap-2 mb-3 text-sm text-gray-600">
          <div class="flex items-center gap-1.5">
            <UsersIcon class="w-3.5 h-3.5 text-blue-500" />
            {{ org.user_count }} users
            <span v-if="org.user_limit > 0" class="text-gray-400">/ {{ org.user_limit }}</span>
          </div>
          <div class="flex items-center gap-1.5">
            <ServerIcon class="w-3.5 h-3.5 text-green-500" />
            {{ org.nas_count }} NAS
          </div>
          <div v-if="org.email" class="flex items-center gap-1.5 col-span-2 truncate">
            <EnvelopeIcon class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" />
            <span class="truncate text-xs">{{ org.email }}</span>
          </div>
        </div>

        <!-- Capacity bar -->
        <div v-if="org.user_limit > 0" class="mb-3">
          <div class="flex justify-between text-xs text-gray-500 mb-1">
            <span>Capacity</span>
            <span>{{ Math.round((org.user_count / org.user_limit) * 100) }}%</span>
          </div>
          <div class="w-full bg-gray-100 rounded-full h-1.5">
            <div class="h-1.5 rounded-full"
              :class="(org.user_count/org.user_limit) > 0.9 ? 'bg-red-500' : 'bg-blue-500'"
              :style="{ width: Math.min((org.user_count/org.user_limit)*100, 100) + '%' }">
            </div>
          </div>
        </div>

        <div v-if="authStore.isSuperAdmin" class="flex gap-2 pt-3 border-t border-gray-100">
          <button @click="viewStats(org)" class="flex-1 btn-secondary text-xs py-1.5">Stats</button>
          <button @click="openEdit(org)" class="flex-1 btn-secondary text-xs py-1.5">Edit</button>
          <button @click="removeOrg(org)" class="p-1.5 rounded-lg border border-red-200 text-red-500 hover:bg-red-50">
            <TrashIcon class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>

    <!-- Stats drawer -->
    <div v-if="statsOrg && stats" class="card">
      <div class="flex items-center justify-between mb-4">
        <h3 class="font-semibold">{{ statsOrg.name }} — Statistics</h3>
        <button @click="statsOrg = null" class="btn-secondary text-xs py-1.5">Close</button>
      </div>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div class="bg-blue-50 rounded-xl p-4 text-center">
          <p class="text-2xl font-bold text-blue-700">{{ stats.total_users }}</p>
          <p class="text-xs text-gray-500 mt-1">Total Users</p>
        </div>
        <div class="bg-green-50 rounded-xl p-4 text-center">
          <p class="text-2xl font-bold text-green-700">{{ stats.active_users }}</p>
          <p class="text-xs text-gray-500 mt-1">Active</p>
        </div>
        <div class="bg-purple-50 rounded-xl p-4 text-center">
          <p class="text-2xl font-bold text-purple-700">{{ stats.active_sessions }}</p>
          <p class="text-xs text-gray-500 mt-1">Sessions Now</p>
        </div>
        <div class="bg-emerald-50 rounded-xl p-4 text-center">
          <p class="text-2xl font-bold text-emerald-700">${{ stats.month_revenue?.toFixed(2) }}</p>
          <p class="text-xs text-gray-500 mt-1">This Month</p>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Organization</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Name <span class="text-red-500">*</span></label>
              <input v-model="form.name" type="text" class="form-input" required placeholder="My ISP Branch" />
            </div>
            <div>
              <label class="form-label">Slug <span class="text-red-500">*</span></label>
              <input v-model="form.slug" type="text" class="form-input" required placeholder="my-isp-branch" />
            </div>
          </div>
          <div>
            <label class="form-label">Email</label>
            <input v-model="form.email" type="email" class="form-input" placeholder="admin@isp.local" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Phone</label>
              <input v-model="form.phone" type="tel" class="form-input" placeholder="+263..." />
            </div>
            <div>
              <label class="form-label">Max Users (0=unlimited)</label>
              <input v-model.number="form.user_limit" type="number" min="0" class="form-input" />
            </div>
          </div>
          <div>
            <label class="form-label">Address</label>
            <input v-model="form.address" type="text" class="form-input" placeholder="Physical address" />
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="org_active" v-model="form.is_active" class="rounded" />
            <label for="org_active" class="text-sm text-gray-700">Organization is active</label>
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
import { ref, reactive, computed, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { useAuthStore } from '@/store/auth'
import { orgsAPI } from '@/api'
import { PlusIcon, XMarkIcon, TrashIcon, UsersIcon, ServerIcon, EnvelopeIcon, BuildingOffice2Icon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const editing = ref(null)
const orgs = ref([])
const statsOrg = ref(null)
const stats = ref(null)
const form = reactive({ name: '', slug: '', email: '', phone: '', address: '', user_limit: 0, is_active: true })

const totalUsers = computed(() => orgs.value.reduce((s, o) => s + (o.user_count || 0), 0))
const totalNAS = computed(() => orgs.value.reduce((s, o) => s + (o.nas_count || 0), 0))

async function load() {
  loading.value = true
  try {
    const { data } = await orgsAPI.list()
    orgs.value = data.data || []
  } catch { } finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', slug: '', email: '', phone: '', address: '', user_limit: 0, is_active: true })
  showModal.value = true
}

function openEdit(org) {
  editing.value = org.id
  Object.assign(form, { name: org.name, slug: org.slug, email: org.email || '', phone: org.phone || '', address: org.address || '', user_limit: org.user_limit, is_active: org.is_active })
  showModal.value = true
}

async function viewStats(org) {
  statsOrg.value = org
  try {
    const { data } = await orgsAPI.stats(org.id)
    stats.value = data
  } catch { }
}

async function save() {
  saving.value = true
  try {
    if (editing.value) { await orgsAPI.update(editing.value, form); toast.success('Updated') }
    else { await orgsAPI.create(form); toast.success('Created') }
    showModal.value = false; load()
  } catch (err) { toast.error(err.response?.data?.error || 'Save failed') }
  finally { saving.value = false }
}

async function removeOrg(org) {
  if (!confirm(`Delete "${org.name}"? This unlinks all users and NAS.`)) return
  try { await orgsAPI.delete(org.id); toast.success('Deleted'); load() }
  catch { toast.error('Failed') }
}

onMounted(load)
</script>
