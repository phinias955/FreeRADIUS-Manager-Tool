<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <transition name="slide-up" appear>
      <div class="modal max-w-sm">
        <div class="modal-header">
          <h2 class="text-lg font-semibold text-gray-900">Reset Password</h2>
          <button @click="$emit('close')" class="p-1.5 hover:bg-gray-100 rounded-lg">
            <XMarkIcon class="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <form @submit.prevent="submit" class="modal-body space-y-4">
          <p class="text-sm text-gray-600">
            Reset password for <span class="font-semibold text-gray-900">{{ user?.username }}</span>
          </p>

          <div>
            <label class="form-label">New Password *</label>
            <div class="relative">
              <input
                v-model="form.new_password"
                :type="showPw ? 'text' : 'password'"
                class="form-input pr-9"
                required
              />
              <button type="button" class="absolute right-2 top-2 text-gray-400" @click="showPw = !showPw">
                <EyeIcon v-if="!showPw" class="w-4 h-4" />
                <EyeSlashIcon v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <div>
            <label class="form-label">Confirm Password *</label>
            <input v-model="confirm" :type="showPw ? 'text' : 'password'" class="form-input" required />
          </div>

          <div v-if="error" class="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
            {{ error }}
          </div>

          <PasswordStrength :password="form.new_password" />
        </form>

        <div class="modal-footer">
          <button @click="$emit('close')" class="btn-secondary">Cancel</button>
          <button @click="submit" class="btn-primary" :disabled="saving">
            <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
            Reset Password
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { radiusUsersAPI } from '@/api'
import { XMarkIcon, EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'
import PasswordStrength from '@/components/common/PasswordStrength.vue'

const props = defineProps({ user: Object })
const emit = defineEmits(['close', 'saved'])

const form = ref({ new_password: '' })
const confirm = ref('')
const showPw = ref(false)
const saving = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  if (form.value.new_password !== confirm.value) {
    error.value = 'Passwords do not match'
    return
  }
  saving.value = true
  try {
    await radiusUsersAPI.resetPassword(props.user.id, form.value)
    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || 'Failed to reset password'
  } finally {
    saving.value = false
  }
}
</script>
