<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Hotspot Zones</h1>
        <p class="text-sm text-gray-500 mt-0.5">Manage physical locations and network zones</p>
      </div>
      <div class="flex gap-2">
        <a href="/portal" target="_blank" class="btn-secondary text-sm">
          <ArrowTopRightOnSquareIcon class="w-4 h-4" />
          User Portal
        </a>
        <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
          <PlusIcon class="w-4 h-4" />
          New Zone
        </button>
      </div>
    </div>

    <!-- Zone cards -->
    <div v-if="loading" class="text-center py-16 text-gray-400">Loading…</div>
    <div v-else-if="!zones.length" class="card text-center py-16 text-gray-400">
      <MapPinIcon class="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p class="font-medium">No zones configured</p>
      <p class="text-sm mt-1">Create zones to group NAS devices by location</p>
    </div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-for="zone in zones" :key="zone.id" class="card cursor-pointer hover:shadow-md transition-shadow"
        @click="viewZone(zone)" :class="!zone.is_active ? 'opacity-60' : ''">
        <div class="flex items-start justify-between mb-3">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl bg-blue-100 flex items-center justify-center">
              <MapPinIcon class="w-5 h-5 text-blue-600" />
            </div>
            <div>
              <h3 class="font-semibold text-gray-900">{{ zone.name }}</h3>
              <p class="text-xs text-gray-500 mt-0.5">{{ zone.location || 'No location set' }}</p>
            </div>
          </div>
          <span class="badge" :class="zone.is_active ? 'badge-green' : 'badge-gray'">
            {{ zone.is_active ? 'Active' : 'Inactive' }}
          </span>
        </div>

        <div class="grid grid-cols-2 gap-3 mb-3">
          <div class="bg-gray-50 rounded-xl p-3 text-center">
            <p class="text-2xl font-bold text-blue-600">{{ zone.nas_count }}</p>
            <p class="text-xs text-gray-500 mt-0.5">NAS Devices</p>
          </div>
          <div class="bg-gray-50 rounded-xl p-3 text-center">
            <p class="text-2xl font-bold text-green-600">{{ zone.active_users }}</p>
            <p class="text-xs text-gray-500 mt-0.5">Active Users</p>
          </div>
        </div>

        <div class="flex gap-2 pt-3 border-t border-gray-100" v-if="authStore.isAdmin" @click.stop>
          <button @click="openEdit(zone)" class="flex-1 btn-secondary text-xs py-1.5">Edit</button>
          <button @click="assignNAS(zone)" class="flex-1 btn-secondary text-xs py-1.5">Assign NAS</button>
          <button @click="removeZone(zone)" class="p-1.5 rounded-lg border border-red-200 text-red-500 hover:bg-red-50">
            <TrashIcon class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>

    <!-- Zone detail panel -->
    <div v-if="selectedZone" class="card">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h3 class="font-semibold text-gray-900">{{ selectedZone.name }} — Details</h3>
          <p class="text-xs text-gray-500 mt-0.5">{{ selectedZone.location }}</p>
        </div>
        <button @click="selectedZone = null" class="btn-secondary text-xs py-1.5">Close</button>
      </div>
      <div v-if="zoneStats">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <!-- Devices table -->
          <div>
            <h4 class="text-sm font-medium text-gray-700 mb-2">NAS Devices</h4>
            <table class="table text-sm">
              <thead>
                <tr>
                  <th>Device</th>
                  <th>Status</th>
                  <th>Users</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="!zoneStats.devices?.length">
                  <td colspan="3" class="text-center py-4 text-gray-400">No devices in zone</td>
                </tr>
                <tr v-for="d in zoneStats.devices" :key="d.id">
                  <td>
                    <p class="font-medium text-xs">{{ d.shortname || d.nasname }}</p>
                    <p class="text-xs text-gray-400 font-mono">{{ d.nasname }}</p>
                  </td>
                  <td>
                    <span class="flex items-center gap-1 text-xs">
                      <span class="w-2 h-2 rounded-full"
                        :class="d.ping_status === 'up' ? 'bg-green-500' : d.ping_status === 'down' ? 'bg-red-500' : 'bg-gray-400'">
                      </span>
                      {{ d.ping_status || 'unknown' }}
                    </span>
                  </td>
                  <td class="font-semibold">{{ d.active_users }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <!-- Daily traffic -->
          <div>
            <h4 class="text-sm font-medium text-gray-700 mb-2">Daily Traffic (7 days)</h4>
            <div v-if="!zoneStats.daily_stats?.length" class="text-center py-8 text-gray-400 text-sm">No data</div>
            <div v-else class="space-y-2">
              <div v-for="day in zoneStats.daily_stats" :key="day.day" class="flex items-center gap-3">
                <span class="text-xs text-gray-500 w-20">{{ day.day }}</span>
                <div class="flex-1 bg-gray-100 rounded-full h-2">
                  <div class="h-2 bg-blue-500 rounded-full" :style="{ width: barWidth(day.total_mb) + '%' }"></div>
                </div>
                <span class="text-xs text-gray-700 w-20 text-right">{{ formatMB(day.total_mb) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Zone</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div>
            <label class="form-label">Zone Name <span class="text-red-500">*</span></label>
            <input v-model="form.name" type="text" class="form-input" required placeholder="e.g. Main Office" />
          </div>
          <div>
            <label class="form-label">Location / Address</label>
            <input v-model="form.location" type="text" class="form-input" placeholder="e.g. Floor 3, CBD" />
          </div>
          <div>
            <label class="form-label">Description</label>
            <input v-model="form.description" type="text" class="form-input" placeholder="Optional" />
          </div>
          <div>
            <label class="form-label">Max Clients (0 = unlimited)</label>
            <input v-model.number="form.max_clients" type="number" min="0" class="form-input" />
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="zone_active" v-model="form.is_active" class="rounded" />
            <label for="zone_active" class="text-sm text-gray-700">Zone is active</label>
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

    <!-- Assign NAS Modal -->
    <div v-if="showAssignNAS" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-sm">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">Assign NAS to {{ assignZone?.name }}</h3>
          <button @click="showAssignNAS = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <div class="p-6 space-y-3">
          <div v-for="nas in allNAS" :key="nas.id" class="flex items-center justify-between p-3 rounded-xl border"
            :class="nas.zone_id === assignZone?.id ? 'border-blue-300 bg-blue-50' : 'border-gray-200'">
            <div>
              <p class="text-sm font-medium">{{ nas.shortname || nas.nasname }}</p>
              <p class="text-xs text-gray-400 font-mono">{{ nas.nasname }}</p>
            </div>
            <button @click="doAssignNAS(nas.id, nas.zone_id === assignZone?.id ? null : assignZone?.id)"
              class="text-xs px-3 py-1.5 rounded-lg"
              :class="nas.zone_id === assignZone?.id ? 'bg-red-100 text-red-600' : 'bg-blue-100 text-blue-700'">
              {{ nas.zone_id === assignZone?.id ? 'Remove' : 'Add' }}
            </button>
          </div>
          <div v-if="!allNAS.length" class="text-center py-4 text-gray-400 text-sm">No NAS devices found</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { useAuthStore } from '@/store/auth'
import { zonesAPI, nasAPI } from '@/api'
import { PlusIcon, XMarkIcon, TrashIcon, MapPinIcon, ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const showAssignNAS = ref(false)
const editing = ref(null)
const zones = ref([])
const allNAS = ref([])
const selectedZone = ref(null)
const zoneStats = ref(null)
const assignZone = ref(null)

const form = reactive({ name: '', location: '', description: '', max_clients: 0, is_active: true })

function formatMB(mb) {
  if (!mb) return '0 MB'
  if (mb >= 1024) return (mb / 1024).toFixed(1) + ' GB'
  return mb.toFixed(0) + ' MB'
}

function barWidth(mb) {
  if (!zoneStats.value?.daily_stats?.length) return 0
  const max = Math.max(...zoneStats.value.daily_stats.map(d => d.total_mb))
  return max > 0 ? (mb / max) * 100 : 0
}

async function load() {
  loading.value = true
  try {
    const { data } = await zonesAPI.list()
    zones.value = data.data || []
  } catch { /* silent */ } finally { loading.value = false }
}

async function viewZone(zone) {
  selectedZone.value = zone
  try {
    const { data } = await zonesAPI.stats(zone.id)
    zoneStats.value = data
  } catch { /* silent */ }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', location: '', description: '', max_clients: 0, is_active: true })
  showModal.value = true
}

function openEdit(zone) {
  editing.value = zone.id
  Object.assign(form, { name: zone.name, location: zone.location || '', description: zone.description || '', max_clients: zone.max_clients, is_active: zone.is_active })
  showModal.value = true
}

async function assignNAS(zone) {
  assignZone.value = zone
  try {
    const { data } = await nasAPI.list({ limit: 100 })
    allNAS.value = data.data || []
  } catch { /* silent */ }
  showAssignNAS.value = true
}

async function doAssignNAS(nasId, zoneId) {
  try {
    await zonesAPI.assignNAS({ nas_id: nasId, zone_id: zoneId })
    toast.success('NAS zone updated')
    const { data } = await nasAPI.list({ limit: 100 })
    allNAS.value = data.data || []
    load()
  } catch { toast.error('Failed') }
}

async function save() {
  saving.value = true
  try {
    if (editing.value) {
      await zonesAPI.update(editing.value, form)
      toast.success('Zone updated')
    } else {
      await zonesAPI.create(form)
      toast.success('Zone created')
    }
    showModal.value = false
    load()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Save failed')
  } finally { saving.value = false }
}

async function removeZone(zone) {
  if (!confirm(`Delete zone "${zone.name}"? All NAS devices will be unlinked.`)) return
  try {
    await zonesAPI.delete(zone.id)
    toast.success('Zone deleted')
    if (selectedZone.value?.id === zone.id) selectedZone.value = null
    load()
  } catch { toast.error('Failed') }
}

onMounted(load)
</script>
