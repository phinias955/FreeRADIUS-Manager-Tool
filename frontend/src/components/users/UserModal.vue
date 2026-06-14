<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <transition name="slide-up" appear>
      <div class="modal">
        <div class="modal-header">
          <h2 class="text-lg font-semibold text-gray-900">
            {{ isEditing ? 'Edit RADIUS User' : 'Create RADIUS User' }}
          </h2>
          <button @click="$emit('close')" class="p-1.5 hover:bg-gray-100 rounded-lg">
            <XMarkIcon class="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <form @submit.prevent="submit" class="modal-body space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Username *</label>
              <input v-model="form.username" :disabled="isEditing" type="text" class="form-input" required />
            </div>
            <div v-if="!isEditing">
              <label class="form-label">Password *</label>
              <div class="relative">
                <input
                  v-model="form.password"
                  :type="showPw ? 'text' : 'password'"
                  class="form-input pr-9"
                  required
                />
                <button type="button" class="absolute right-2 top-2 text-gray-400" @click="showPw = !showPw">
                  <EyeIcon v-if="!showPw" class="w-4 h-4" />
                  <EyeSlashIcon v-else class="w-4 h-4" />
                </button>
              </div>
              <p class="text-xs text-gray-500 mt-1">Min 12 chars, uppercase, lowercase, number, special</p>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Full Name</label>
              <input v-model="form.full_name" type="text" class="form-input" />
            </div>
            <div>
              <label class="form-label">Email</label>
              <input v-model="form.email" type="email" class="form-input" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Department</label>
              <input v-model="form.department" type="text" class="form-input" />
            </div>
            <div>
              <label class="form-label">Account Expiry</label>
              <input v-model="form.account_expiry" type="date" class="form-input" />
            </div>
          </div>

          <!-- Device limit slider -->
          <div>
            <label class="form-label">
              Device Limit
              <span class="ml-2 font-semibold text-blue-600">{{ form.device_limit }}</span>
            </label>
            <div class="flex items-center gap-3">
              <span class="text-xs text-gray-500">1</span>
              <input
                v-model.number="form.device_limit"
                type="range"
                min="1"
                max="20"
                class="flex-1 accent-blue-600"
              />
              <span class="text-xs text-gray-500">20</span>
            </div>
            <div class="flex justify-between mt-1">
              <span class="text-xs text-gray-400">Single device</span>
              <span class="text-xs text-gray-400">Up to 20 devices</span>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <input v-model="form.force_password_change" type="checkbox" id="force-pw" class="rounded" />
            <label for="force-pw" class="text-sm text-gray-700">Force password change on next login</label>
          </div>

          <div v-if="error" class="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
            {{ error }}
          </div>
        </form>

        <div class="modal-footer">
          <button @click="$emit('close')" class="btn-secondary">Cancel</button>
          <button @click="submit" class="btn-primary" :disabled="saving">
            <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
            {{ isEditing ? 'Save Changes' : 'Create User' }}
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { radiusUsersAPI } from '@/api'
import { XMarkIcon, EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'

const props = defineProps({ user: Object })
const emit = defineEmits(['close', 'saved'])
const toast = useToast()

const isEditing = computed(() => !!props.user?.id)
const saving = ref(false)
const showPw = ref(false)
const error = ref('')

const form = ref({
  username: '',
  password: '',
  full_name: '',
  email: '',
  department: '',
  device_limit: 1,
  account_expiry: '',
  force_password_change: false,
})

watch(() => props.user, (u) => {
  if (u) {
    form.value = {
      username: u.username || '',
      full_name: u.full_name || '',
      email: u.email || '',
      department: u.department || '',
      device_limit: u.device_limit || 1,
      account_expiry: u.account_expiry ? u.account_expiry.slice(0, 10) : '',
      force_password_change: u.force_password_change || false,
    }
  }
}, { immediate: true })

async function submit() {
  error.value = ''
  saving.value = true
  try {
    if (isEditing.value) {
      await radiusUsersAPI.update(props.user.id, {
        full_name: form.value.full_name,
        email: form.value.email,
        department: form.value.department,
        device_limit: form.value.device_limit,
        account_expiry: form.value.account_expiry || null,
        force_password_change: form.value.force_password_change,
      })
      toast.success('User updated successfully')
    } else {
      await radiusUsersAPI.create(form.value)
      toast.success('User created successfully')
    }
    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || 'Failed to save user'
  } finally {
    saving.value = false
  }
}
</script>
