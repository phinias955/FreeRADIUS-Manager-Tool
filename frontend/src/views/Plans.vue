<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">User Plans</h1>
        <p class="text-sm text-gray-500 mt-0.5">Subscription plans and packages assigned to users</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
        <PlusIcon class="w-4 h-4" />
        New Plan
      </button>
    </div>

    <!-- Plan cards -->
    <div v-if="loading" class="text-center py-16 text-gray-400">Loading…</div>
    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
      <div
        v-for="p in plans" :key="p.id"
        class="card p-5 flex flex-col gap-3 relative"
        :class="!p.is_active ? 'opacity-60' : ''"
      >
        <!-- Popular badge -->
        <div v-if="p.user_count > 0" class="absolute -top-2 -right-2 bg-blue-600 text-white text-xs px-2 py-0.5 rounded-full font-medium">
          {{ p.user_count }} users
        </div>

        <div>
          <h3 class="font-bold text-gray-900">{{ p.name }}</h3>
          <p class="text-xs text-gray-500 mt-0.5">{{ p.description || 'No description' }}</p>
        </div>

        <div class="text-3xl font-bold text-blue-600">
          {{ p.price > 0 ? p.currency + ' ' + p.price.toFixed(2) : 'Free' }}
          <span class="text-sm font-normal text-gray-400">/ {{ p.validity_days }}d</span>
        </div>

        <ul class="space-y-1 text-sm text-gray-600 flex-1">
          <li class="flex items-center gap-2">
            <CheckIcon class="w-4 h-4 text-green-500 flex-shrink-0" />
            {{ p.data_limit_mb ? formatMB(p.data_limit_mb) + ' Data' : 'Unlimited Data' }}
          </li>
          <li class="flex items-center gap-2">
            <CheckIcon class="w-4 h-4 text-green-500 flex-shrink-0" />
            {{ p.max_devices }} Device{{ p.max_devices > 1 ? 's' : '' }}
          </li>
          <li v-if="p.bandwidth_name" class="flex items-center gap-2">
            <CheckIcon class="w-4 h-4 text-green-500 flex-shrink-0" />
            {{ p.bandwidth_name }}
          </li>
        </ul>

        <div v-if="authStore.isAdmin" class="flex gap-2 pt-2 border-t border-gray-100">
          <button @click="openEdit(p)" class="flex-1 btn-secondary text-xs py-1.5">Edit</button>
          <button @click="remove(p)" class="flex-1 text-xs py-1.5 rounded-lg border border-red-200 text-red-600 hover:bg-red-50">Delete</button>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-lg">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Plan</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div class="col-span-2">
              <label class="form-label">Plan Name <span class="text-red-500">*</span></label>
              <input v-model="form.name" type="text" class="form-input" required placeholder="e.g. Standard 20GB" />
            </div>
            <div>
              <label class="form-label">Price</label>
              <input v-model.number="form.price" type="number" min="0" step="0.01" class="form-input" placeholder="0.00" />
            </div>
            <div>
              <label class="form-label">Currency</label>
              <input v-model="form.currency" type="text" class="form-input" maxlength="3" placeholder="USD" />
            </div>
            <div>
              <label class="form-label">Data Limit (MB)</label>
              <input v-model.number="form.data_limit_mb" type="number" min="0" class="form-input" placeholder="Leave empty = unlimited" />
            </div>
            <div>
              <label class="form-label">Validity (Days)</label>
              <input v-model.number="form.validity_days" type="number" min="1" class="form-input" placeholder="30" />
            </div>
            <div>
              <label class="form-label">Max Devices</label>
              <input v-model.number="form.max_devices" type="number" min="1" class="form-input" />
            </div>
            <div>
              <label class="form-label">Speed Profile</label>
              <select v-model="form.bandwidth_profile_id" class="form-input">
                <option :value="null">None (unlimited speed)</option>
                <option v-for="bp in bandwidthProfiles" :key="bp.id" :value="bp.id">{{ bp.name }}</option>
              </select>
            </div>
            <div class="col-span-2">
              <label class="form-label">Description</label>
              <input v-model="form.description" type="text" class="form-input" placeholder="Optional description" />
            </div>
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="plan_active" v-model="form.is_active" class="rounded" />
            <label for="plan_active" class="text-sm text-gray-700">Active</label>
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
import { plansAPI, bandwidthAPI } from '@/api'
import { PlusIcon, XMarkIcon, CheckIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const editing = ref(null)
const plans = ref([])
const bandwidthProfiles = ref([])

const form = reactive({
  name: '', description: '', price: 0, currency: 'USD',
  data_limit_mb: null, validity_days: 30, max_devices: 1,
  bandwidth_profile_id: null, is_active: true,
})

function formatMB(mb) {
  if (mb >= 1024) return (mb / 1024).toFixed(0) + ' GB'
  return mb + ' MB'
}

async function load() {
  loading.value = true
  try {
    const [{ data: p }, { data: b }] = await Promise.all([plansAPI.list(), bandwidthAPI.list()])
    plans.value = p.data || []
    bandwidthProfiles.value = b.data || []
  } catch { /* silent */ } finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', description: '', price: 0, currency: 'USD', data_limit_mb: null, validity_days: 30, max_devices: 1, bandwidth_profile_id: null, is_active: true })
  showModal.value = true
}

function openEdit(p) {
  editing.value = p.id
  Object.assign(form, { ...p })
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    const payload = { ...form }
    if (!payload.data_limit_mb) payload.data_limit_mb = null
    if (editing.value) {
      await plansAPI.update(editing.value, payload)
      toast.success('Plan updated')
    } else {
      await plansAPI.create(payload)
      toast.success('Plan created')
    }
    showModal.value = false
    load()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Save failed')
  } finally { saving.value = false }
}

async function remove(p) {
  if (!confirm(`Delete plan "${p.name}"? Users assigned to it will lose the plan.`)) return
  try {
    await plansAPI.delete(p.id)
    toast.success('Plan deleted')
    load()
  } catch (err) { toast.error(err.response?.data?.error || 'Failed') }
}

onMounted(load)
</script>
