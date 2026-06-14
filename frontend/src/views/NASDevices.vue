<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">NAS Devices</h1>
        <p class="text-sm text-gray-500 mt-0.5">Manage RADIUS clients (routers, APs, VPN gateways)</p>
      </div>
      <div class="flex items-center gap-2" v-if="authStore.isAdmin">
        <button @click="openDiscover" class="btn-secondary">
          <MagnifyingGlassIcon class="w-4 h-4" />
          Discover
        </button>
        <button @click="openCreate" class="btn-primary">
          <PlusIcon class="w-4 h-4" />
          Add Device
        </button>
      </div>
    </div>

    <!-- Device grid -->
    <div v-if="loading" class="h-64 flex items-center justify-center">
      <span class="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full spinner"></span>
    </div>

    <div v-else-if="!devices.length" class="card text-center py-16">
      <ServerIcon class="w-12 h-12 text-gray-300 mx-auto mb-4" />
      <p class="text-gray-500 font-medium">No NAS devices configured</p>
      <p class="text-sm text-gray-400 mt-1">Add routers, access points, or VPN gateways</p>
      <button v-if="authStore.isAdmin" @click="openCreate" class="btn-primary mt-4">
        <PlusIcon class="w-4 h-4" />
        Add First Device
      </button>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="device in devices"
        :key="device.id"
        class="card hover:shadow-md transition-shadow"
      >
        <div class="flex items-start justify-between mb-3">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg flex items-center justify-center" :class="deviceIconBg(device.type)">
              <ServerIcon class="w-5 h-5" :class="deviceIconColor(device.type)" />
            </div>
            <div>
              <p class="font-semibold text-gray-900">{{ device.shortname || device.nasname }}</p>
              <p class="text-xs text-gray-500 font-mono">{{ device.nasname }}</p>
            </div>
          </div>
          <span :class="device.status === 'active' ? 'badge-green' : 'badge-gray'" class="badge">
            {{ device.status }}
          </span>
        </div>

        <div class="space-y-1.5 text-sm text-gray-600 mb-4">
          <div class="flex items-center gap-2">
            <span class="w-16 text-xs text-gray-400">Type</span>
            <span class="capitalize">{{ device.type || 'other' }}</span>
          </div>
          <div class="flex items-center gap-2" v-if="device.description">
            <span class="w-16 text-xs text-gray-400">Info</span>
            <span class="truncate">{{ device.description }}</span>
          </div>
          <div class="flex items-center gap-2" v-if="device.ports">
            <span class="w-16 text-xs text-gray-400">Ports</span>
            <span>{{ device.ports }}</span>
          </div>
        </div>

        <div class="flex items-center gap-2 pt-3 border-t border-gray-100">
          <button
            @click="testDevice(device)"
            class="btn-secondary flex-1 justify-center py-1.5 text-xs"
            :disabled="testing === device.id"
          >
            <span v-if="testing === device.id" class="w-3 h-3 border border-gray-400 border-t-transparent rounded-full spinner"></span>
            <SignalIcon v-else class="w-3.5 h-3.5" />
            Test
          </button>
          <template v-if="authStore.isAdmin">
            <button @click="openEdit(device)" class="btn-secondary py-1.5 px-3">
              <PencilIcon class="w-3.5 h-3.5" />
            </button>
            <button @click="confirmDelete(device)" class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg">
              <TrashIcon class="w-3.5 h-3.5" />
            </button>
          </template>
        </div>

        <!-- Test result -->
        <div v-if="testResults[device.id]" class="mt-3 p-2.5 rounded-lg text-xs"
          :class="testResults[device.id].success ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-red-50 text-red-700 border border-red-200'"
        >
          <p class="font-medium">{{ testResults[device.id].success ? '✓ Reachable' : '✗ Unreachable' }}</p>
          <p class="opacity-75">{{ testResults[device.id].message }}</p>
          <p class="opacity-75">Latency: {{ testResults[device.id].latency_ms?.toFixed(1) }}ms</p>
        </div>

        <!-- Live ping badge -->
        <div class="mt-2 flex items-center gap-2 text-xs" v-if="pingStatus[device.id]">
          <span class="w-2 h-2 rounded-full"
            :class="pingStatus[device.id] === 'up' ? 'bg-green-500' : pingStatus[device.id] === 'down' ? 'bg-red-500' : 'bg-gray-400'">
          </span>
          <span :class="pingStatus[device.id] === 'up' ? 'text-green-600' : pingStatus[device.id] === 'down' ? 'text-red-500' : 'text-gray-400'">
            {{ pingStatus[device.id] === 'up' ? 'Online' : pingStatus[device.id] === 'down' ? 'Offline' : 'Unknown' }}
          </span>
          <span v-if="pingLatency[device.id]" class="text-gray-400">{{ pingLatency[device.id].toFixed(0) }}ms</span>
        </div>
      </div>
    </div>

    <!-- NAS Templates info -->
    <div class="card bg-blue-50 border border-blue-200">
      <h3 class="font-semibold text-blue-900 mb-2">Device Configuration Templates</h3>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div v-for="tmpl in templates" :key="tmpl.name"
          class="bg-white rounded-lg p-3 border border-blue-200 cursor-pointer hover:border-blue-400 transition-colors"
          @click="openCreateFromTemplate(tmpl)"
        >
          <p class="font-medium text-sm text-gray-900">{{ tmpl.name }}</p>
          <p class="text-xs text-gray-500 mt-0.5">{{ tmpl.desc }}</p>
        </div>
      </div>
    </div>

    <!-- Modals -->
    <NASModal v-if="showModal" :device="editingDevice" @close="showModal = false" @saved="onSaved" />
    <DiscoverModal v-if="showDiscover" @close="showDiscover = false" @add="addDiscovered" />
    <ConfirmDialog
      v-if="showDeleteConfirm"
      title="Delete NAS Device"
      :message="`Remove '${editingDevice?.shortname || editingDevice?.nasname}'?`"
      @confirm="deleteDevice"
      @cancel="showDeleteConfirm = false"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '@/store/auth'
import { useToast } from 'vue-toastification'
import { nasAPI, nasStatusAPI } from '@/api'
import { PlusIcon, PencilIcon, TrashIcon, ServerIcon, SignalIcon, MagnifyingGlassIcon } from '@heroicons/vue/24/outline'
import NASModal from '@/components/nas/NASModal.vue'
import DiscoverModal from '@/components/nas/DiscoverModal.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const authStore = useAuthStore()
const toast = useToast()
const devices = ref([])
const loading = ref(false)
const testing = ref(null)
const testResults = ref({})
const pingStatus = ref({})
const pingLatency = ref({})
const showModal = ref(false)
const showDiscover = ref(false)
const showDeleteConfirm = ref(false)
const editingDevice = ref(null)
let pingTimer

const templates = [
  { name: 'MikroTik', desc: 'RouterOS devices', type: 'other', ports: 1812 },
  { name: 'Cisco', desc: 'IOS / ASA / WLC', type: 'cisco', ports: 1812 },
  { name: 'Ubiquiti', desc: 'UniFi APs', type: 'other', ports: 1812 },
  { name: 'pfSense', desc: 'pfSense / OPNsense', type: 'other', ports: 1812 },
]

async function loadDevices() {
  loading.value = true
  try {
    const { data } = await nasAPI.list({ limit: 100 })
    devices.value = data.data || []
  } catch {
    toast.error('Failed to load NAS devices')
  } finally {
    loading.value = false
  }
}

async function testDevice(device) {
  testing.value = device.id
  try {
    const { data } = await nasAPI.test(device.id)
    testResults.value[device.id] = data
  } catch {
    testResults.value[device.id] = { success: false, message: 'Test request failed', latency_ms: 0 }
  } finally {
    testing.value = null
  }
}

function openCreate() { editingDevice.value = null; showModal.value = true }
function openEdit(d) { editingDevice.value = { ...d }; showModal.value = true }
function openDiscover() { showDiscover.value = true }
function confirmDelete(d) { editingDevice.value = d; showDeleteConfirm.value = true }
function openCreateFromTemplate(tmpl) {
  editingDevice.value = { type: tmpl.type, ports: tmpl.ports }
  showModal.value = true
}

async function deleteDevice() {
  showDeleteConfirm.value = false
  try {
    await nasAPI.delete(editingDevice.value.id)
    toast.success('Device deleted')
    loadDevices()
  } catch { toast.error('Failed to delete device') }
}

function onSaved() { showModal.value = false; loadDevices() }
function addDiscovered(ip) { showDiscover.value = false; editingDevice.value = { nasname: ip }; showModal.value = true }

async function loadPingStatus() {
  try {
    const { data } = await nasStatusAPI.status()
    ;(data.data || []).forEach(d => {
      pingStatus.value[d.id] = d.ping_status
      pingLatency.value[d.id] = d.ping_latency_ms
    })
  } catch { /* silent */ }
}

function deviceIconBg(type) {
  return { cisco: 'bg-blue-100', other: 'bg-gray-100' }[type] || 'bg-gray-100'
}
function deviceIconColor(type) {
  return { cisco: 'text-blue-600', other: 'text-gray-600' }[type] || 'text-gray-600'
}

onMounted(() => {
  loadDevices()
  loadPingStatus()
  pingTimer = setInterval(loadPingStatus, 60000)
})
onUnmounted(() => clearInterval(pingTimer))
</script>
