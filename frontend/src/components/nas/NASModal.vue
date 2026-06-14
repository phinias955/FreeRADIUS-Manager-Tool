<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <transition name="slide-up" appear>
      <div class="modal">
        <div class="modal-header">
          <h2 class="text-lg font-semibold text-gray-900">
            {{ isEditing ? 'Edit NAS Device' : 'Add NAS Device' }}
          </h2>
          <button @click="$emit('close')" class="p-1.5 hover:bg-gray-100 rounded-lg">
            <XMarkIcon class="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <form @submit.prevent="submit" class="modal-body space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">IP Address / Hostname *</label>
              <input v-model="form.nasname" type="text" class="form-input" :disabled="isEditing" required
                placeholder="192.168.1.1" />
            </div>
            <div>
              <label class="form-label">Short Name *</label>
              <input v-model="form.shortname" type="text" class="form-input" required placeholder="router-main" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Device Type</label>
              <select v-model="form.type" class="form-select">
                <option value="other">Other</option>
                <option value="cisco">Cisco</option>
                <option value="mikrotik">MikroTik</option>
                <option value="ubiquiti">Ubiquiti</option>
                <option value="pfsense">pfSense</option>
                <option value="juniper">Juniper</option>
                <option value="hp">HP/Aruba</option>
              </select>
            </div>
            <div>
              <label class="form-label">Port</label>
              <input v-model.number="form.ports" type="number" class="form-input" placeholder="1812" />
            </div>
          </div>

          <div>
            <label class="form-label">Shared Secret *</label>
            <div class="relative">
              <input
                v-model="form.secret"
                :type="showSecret ? 'text' : 'password'"
                class="form-input pr-20"
                required
                :placeholder="isEditing ? 'Leave blank to keep current' : 'Min 8 characters'"
              />
              <div class="absolute right-2 top-2 flex gap-1">
                <button type="button" class="text-gray-400 hover:text-gray-600" @click="showSecret = !showSecret">
                  <EyeIcon v-if="!showSecret" class="w-4 h-4" />
                  <EyeSlashIcon v-else class="w-4 h-4" />
                </button>
                <button type="button" class="text-gray-400 hover:text-blue-600" @click="generateSecret" title="Generate">
                  <ArrowPathIcon class="w-4 h-4" />
                </button>
              </div>
            </div>
            <p class="text-xs text-gray-500 mt-1">Minimum 8 characters. Use 32+ for production.</p>
          </div>

          <div>
            <label class="form-label">Description</label>
            <textarea v-model="form.description" class="form-input" rows="2" placeholder="e.g., Main office router, 2nd floor AP"></textarea>
          </div>

          <div v-if="isEditing">
            <label class="form-label">Status</label>
            <select v-model="form.status" class="form-select">
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
          </div>

          <div v-if="error" class="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
            {{ error }}
          </div>
        </form>

        <div class="modal-footer">
          <button @click="$emit('close')" class="btn-secondary">Cancel</button>
          <button @click="submit" class="btn-primary" :disabled="saving">
            <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
            {{ isEditing ? 'Save Changes' : 'Add Device' }}
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { nasAPI } from '@/api'
import { XMarkIcon, EyeIcon, EyeSlashIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'

const props = defineProps({ device: Object })
const emit = defineEmits(['close', 'saved'])
const toast = useToast()

const isEditing = computed(() => !!props.device?.id)
const saving = ref(false)
const showSecret = ref(false)
const error = ref('')

const form = ref({
  nasname: '', shortname: '', type: 'other', ports: null, secret: '',
  server: '', community: '', description: '', status: 'active',
})

watch(() => props.device, (d) => {
  if (d) Object.assign(form.value, { ...form.value, ...d, secret: '' })
}, { immediate: true })

function generateSecret() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*'
  form.value.secret = Array.from({ length: 32 }, () => chars[Math.floor(Math.random() * chars.length)]).join('')
  showSecret.value = true
}

async function submit() {
  error.value = ''
  saving.value = true
  try {
    if (isEditing.value) {
      const payload = { ...form.value }
      if (!payload.secret) delete payload.secret
      await nasAPI.update(props.device.id, payload)
      toast.success('NAS device updated')
    } else {
      await nasAPI.create(form.value)
      toast.success('NAS device added')
    }
    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || 'Failed to save device'
  } finally {
    saving.value = false
  }
}
</script>
