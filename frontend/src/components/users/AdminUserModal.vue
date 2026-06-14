<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <transition name="slide-up" appear>
      <div class="modal">
        <div class="modal-header">
          <h2 class="text-lg font-semibold text-gray-900">
            {{ isEditing ? 'Edit Admin User' : 'Create Admin User' }}
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
            <div>
              <label class="form-label">Full Name</label>
              <input v-model="form.full_name" type="text" class="form-input" />
            </div>
          </div>

          <div>
            <label class="form-label">Email *</label>
            <input v-model="form.email" type="email" class="form-input" required />
          </div>

          <div v-if="!isEditing">
            <label class="form-label">Password *</label>
            <div class="relative">
              <input v-model="form.password" :type="showPw ? 'text' : 'password'" class="form-input pr-9" required />
              <button type="button" class="absolute right-2 top-2 text-gray-400" @click="showPw = !showPw">
                <EyeIcon v-if="!showPw" class="w-4 h-4" />
                <EyeSlashIcon v-else class="w-4 h-4" />
              </button>
            </div>
            <PasswordStrength :password="form.password" class="mt-1" />
          </div>

          <div>
            <label class="form-label">Role *</label>
            <select v-model="form.role" class="form-select" required>
              <option value="operator">Operator (view-only + password reset)</option>
              <option value="admin">Admin (manage users & NAS)</option>
              <option value="super_admin">Super Admin (full access)</option>
            </select>
            <p class="text-xs text-gray-500 mt-1">You cannot assign a role equal to or higher than your own.</p>
          </div>

          <div v-if="isEditing" class="flex items-center gap-2">
            <input v-model="form.is_active" type="checkbox" id="user-active" class="rounded" />
            <label for="user-active" class="text-sm text-gray-700">Account active</label>
          </div>

          <div v-if="error" class="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
            {{ error }}
          </div>
        </form>

        <div class="modal-footer">
          <button @click="$emit('close')" class="btn-secondary">Cancel</button>
          <button @click="submit" class="btn-primary" :disabled="saving">
            <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
            {{ isEditing ? 'Save Changes' : 'Create Admin' }}
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { adminAPI } from '@/api'
import { XMarkIcon, EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'
import PasswordStrength from '@/components/common/PasswordStrength.vue'
import { useToast } from 'vue-toastification'

const props = defineProps({ user: Object })
const emit = defineEmits(['close', 'saved'])
const toast = useToast()

const isEditing = computed(() => !!props.user?.id)
const saving = ref(false)
const showPw = ref(false)
const error = ref('')

const form = ref({ username: '', password: '', email: '', full_name: '', role: 'operator', is_active: true })

watch(() => props.user, (u) => {
  if (u) Object.assign(form.value, { ...form.value, ...u })
}, { immediate: true })

async function submit() {
  error.value = ''
  saving.value = true
  try {
    if (isEditing.value) {
      await adminAPI.update(props.user.id, {
        email: form.value.email,
        full_name: form.value.full_name,
        role: form.value.role,
        is_active: form.value.is_active,
      })
      toast.success('Admin user updated')
    } else {
      await adminAPI.create(form.value)
      toast.success('Admin user created')
    }
    emit('saved')
  } catch (err) {
    error.value = err.response?.data?.error || 'Failed to save'
  } finally {
    saving.value = false
  }
}
</script>
