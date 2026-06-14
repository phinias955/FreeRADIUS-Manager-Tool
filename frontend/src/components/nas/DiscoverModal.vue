<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <transition name="slide-up" appear>
      <div class="modal">
        <div class="modal-header">
          <h2 class="text-lg font-semibold text-gray-900">Discover NAS Devices</h2>
          <button @click="$emit('close')" class="p-1.5 hover:bg-gray-100 rounded-lg">
            <XMarkIcon class="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <div class="modal-body space-y-4">
          <p class="text-sm text-gray-600">
            Scan a subnet for RADIUS-capable devices. The scanner sends test packets to port 1812/UDP.
          </p>

          <div class="flex gap-3">
            <div class="flex-1">
              <label class="form-label">Subnet (CIDR) *</label>
              <input v-model="subnet" type="text" class="form-input" placeholder="192.168.1.0/24" />
            </div>
            <div class="flex items-end">
              <button @click="startScan" class="btn-primary" :disabled="scanning || !subnet">
                <span v-if="scanning" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
                <MagnifyingGlassIcon v-else class="w-4 h-4" />
                {{ scanning ? 'Scanning...' : 'Scan' }}
              </button>
            </div>
          </div>

          <div v-if="scanning" class="text-center py-8">
            <span class="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full spinner inline-block mb-2"></span>
            <p class="text-sm text-gray-600">Scanning network, this may take a moment...</p>
          </div>

          <div v-else-if="results !== null">
            <p class="text-sm font-medium text-gray-700 mb-3">
              Found {{ results.count }} device(s) in {{ subnet }}
            </p>
            <div v-if="results.discovered?.length" class="space-y-2 max-h-64 overflow-y-auto">
              <div
                v-for="d in results.discovered"
                :key="d.ip"
                class="flex items-center justify-between p-3 bg-gray-50 rounded-lg border border-gray-200"
              >
                <div>
                  <p class="font-mono font-medium text-sm">{{ d.ip }}</p>
                  <p class="text-xs text-gray-500">Latency: {{ d.latency?.toFixed(1) }}ms</p>
                </div>
                <button @click="$emit('add', d.ip)" class="btn-primary py-1.5 px-3 text-xs">
                  Add Device
                </button>
              </div>
            </div>
            <div v-else class="text-center py-8 text-gray-400 text-sm">
              No RADIUS devices found in the specified subnet.
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button @click="$emit('close')" class="btn-secondary">Close</button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { nasAPI } from '@/api'
import { XMarkIcon, MagnifyingGlassIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'

defineEmits(['close', 'add'])
const toast = useToast()

const subnet = ref('192.168.1.0/24')
const scanning = ref(false)
const results = ref(null)

async function startScan() {
  scanning.value = true
  results.value = null
  try {
    const { data } = await nasAPI.discover({ subnet: subnet.value })
    results.value = data
  } catch (err) {
    toast.error(err.response?.data?.error || 'Scan failed')
    results.value = { count: 0, discovered: [] }
  } finally {
    scanning.value = false
  }
}
</script>
