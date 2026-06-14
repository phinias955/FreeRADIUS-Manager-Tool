<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Bandwidth Profiles</h1>
        <p class="text-sm text-gray-500 mt-0.5">Speed plans assigned to RADIUS users</p>
      </div>
      <button @click="openCreate" class="btn-primary">
        <PlusIcon class="w-4 h-4" />
        New Profile
      </button>
    </div>

    <!-- Grid of profiles -->
    <div v-if="loading" class="text-center py-16 text-gray-400">Loading…</div>
    <div v-else-if="!profiles.length" class="card p-12 text-center text-gray-400">
      No bandwidth profiles yet. Create one to start limiting user speeds.
    </div>
    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      <div
        v-for="p in profiles" :key="p.id"
        class="card p-5 flex flex-col gap-3"
        :class="!p.is_active ? 'opacity-50' : ''"
      >
        <div class="flex items-start justify-between">
          <div>
            <h3 class="font-semibold text-gray-900 text-sm">{{ p.name }}</h3>
            <p class="text-xs text-gray-500 mt-0.5">{{ p.description || 'No description' }}</p>
          </div>
          <span class="badge" :class="p.is_active ? 'badge-green' : 'badge-gray'">
            {{ p.is_active ? 'Active' : 'Inactive' }}
          </span>
        </div>

        <!-- Speed bars -->
        <div class="space-y-2">
          <div>
            <div class="flex justify-between text-xs text-gray-500 mb-1">
              <span>↑ Upload</span>
              <span class="font-mono font-medium text-gray-700">{{ formatKbps(p.upload_kbps) }}</span>
            </div>
            <div class="h-1.5 bg-gray-100 rounded-full overflow-hidden">
              <div class="h-full bg-blue-500 rounded-full" :style="{ width: speedBar(p.upload_kbps) }"></div>
            </div>
          </div>
          <div>
            <div class="flex justify-between text-xs text-gray-500 mb-1">
              <span>↓ Download</span>
              <span class="font-mono font-medium text-gray-700">{{ formatKbps(p.download_kbps) }}</span>
            </div>
            <div class="h-1.5 bg-gray-100 rounded-full overflow-hidden">
              <div class="h-full bg-green-500 rounded-full" :style="{ width: speedBar(p.download_kbps) }"></div>
            </div>
          </div>
        </div>

        <div class="text-xs text-gray-400 font-mono bg-gray-50 rounded px-2 py-1">
          MikroTik: {{ p.mikrotik_rate_limit || '—' }}
        </div>

        <div class="flex gap-2 mt-auto pt-2 border-t border-gray-100">
          <button @click="openEdit(p)" class="flex-1 btn-secondary text-xs py-1.5">Edit</button>
          <button @click="remove(p)" class="flex-1 text-xs py-1.5 rounded-lg border border-red-200 text-red-600 hover:bg-red-50">Delete</button>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-lg">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Bandwidth Profile</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div>
            <label class="form-label">Profile Name <span class="text-red-500">*</span></label>
            <input v-model="form.name" type="text" class="form-input" placeholder="e.g. Premium 10M/20M" required />
          </div>
          <div>
            <label class="form-label">Description</label>
            <input v-model="form.description" type="text" class="form-input" placeholder="Optional description" />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Upload (Kbps) <span class="text-red-500">*</span></label>
              <input v-model.number="form.upload_kbps" type="number" min="1" class="form-input" required />
              <p class="text-xs text-gray-400 mt-1">{{ formatKbps(form.upload_kbps) }}</p>
            </div>
            <div>
              <label class="form-label">Download (Kbps) <span class="text-red-500">*</span></label>
              <input v-model.number="form.download_kbps" type="number" min="1" class="form-input" required />
              <p class="text-xs text-gray-400 mt-1">{{ formatKbps(form.download_kbps) }}</p>
            </div>
          </div>
          <div>
            <label class="form-label">MikroTik Rate-Limit (auto-generated if blank)</label>
            <input v-model="form.mikrotik_rate_limit" type="text" class="form-input font-mono" placeholder="e.g. 5M/10M" />
            <p class="text-xs text-gray-400 mt-1">Auto: {{ autoMikrotik }}</p>
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="is_active" v-model="form.is_active" class="rounded" />
            <label for="is_active" class="text-sm text-gray-700">Active (users can be assigned this profile)</label>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              {{ editing ? 'Update' : 'Create' }} Profile
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
import { bandwidthAPI } from '@/api'
import { PlusIcon, XMarkIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const editing = ref(null)
const profiles = ref([])

const form = reactive({
  name: '', description: '', upload_kbps: 1024, download_kbps: 2048,
  burst_upload_kbps: 0, burst_download_kbps: 0, mikrotik_rate_limit: '', is_active: true,
})

const MAX_KBPS = 102400 // 100 Mbps reference for bar width

function formatKbps(kbps) {
  if (!kbps) return '—'
  if (kbps >= 1000) return (kbps / 1000).toFixed(kbps % 1000 === 0 ? 0 : 1) + ' Mbps'
  return kbps + ' Kbps'
}

function speedBar(kbps) {
  return Math.min(100, (kbps / MAX_KBPS) * 100).toFixed(1) + '%'
}

const autoMikrotik = computed(() => {
  const u = form.upload_kbps
  const d = form.download_kbps
  const fmt = k => (k >= 1000 && k % 1000 === 0) ? `${k/1000}M` : `${k}k`
  return `${fmt(u)}/${fmt(d)}`
})

async function load() {
  loading.value = true
  try {
    const { data } = await bandwidthAPI.list()
    profiles.value = data.data || []
  } catch { /* silent */ } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', description: '', upload_kbps: 1024, download_kbps: 2048, burst_upload_kbps: 0, burst_download_kbps: 0, mikrotik_rate_limit: '', is_active: true })
  showModal.value = true
}

function openEdit(p) {
  editing.value = p.id
  Object.assign(form, {
    name: p.name,
    description: p.description || '',
    upload_kbps: p.upload_kbps,
    download_kbps: p.download_kbps,
    burst_upload_kbps: p.burst_upload_kbps || 0,
    burst_download_kbps: p.burst_download_kbps || 0,
    mikrotik_rate_limit: p.mikrotik_rate_limit || '',
    is_active: p.is_active,
  })
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) {
      await bandwidthAPI.update(editing.value, form)
      toast.success('Profile updated')
    } else {
      await bandwidthAPI.create(form)
      toast.success('Profile created')
    }
    showModal.value = false
    load()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Save failed')
  } finally {
    saving.value = false
  }
}

async function remove(p) {
  if (!confirm(`Delete profile "${p.name}"? Users assigned to it will lose their speed limits.`)) return
  try {
    await bandwidthAPI.delete(p.id)
    toast.success('Profile deleted')
    load()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed')
  }
}

onMounted(load)
</script>
