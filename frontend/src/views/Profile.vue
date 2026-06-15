<template>
  <div class="space-y-6 max-w-3xl">
    <div class="page-header">
      <div>
        <h1 class="page-title">My Profile</h1>
        <p class="text-sm text-gray-500 mt-0.5">Update your account details and password</p>
      </div>
    </div>

    <div v-if="loading" class="h-48 flex items-center justify-center">
      <span class="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full spinner"></span>
    </div>

    <template v-else>
      <!-- Account summary -->
      <div class="card flex items-center gap-4">
        <div class="w-14 h-14 rounded-full bg-blue-100 flex items-center justify-center flex-shrink-0">
          <span class="text-blue-700 font-bold text-lg">{{ initials }}</span>
        </div>
        <div>
          <p class="font-semibold text-gray-900">{{ profile.full_name || profile.username }}</p>
          <p class="text-sm text-gray-500">@{{ profile.username }}</p>
          <span class="inline-block mt-1 text-xs font-medium capitalize px-2 py-0.5 rounded-full bg-blue-50 text-blue-700">
            {{ profile.role?.replace('_', ' ') }}
          </span>
        </div>
        <div class="ml-auto text-right text-xs text-gray-400 hidden sm:block">
          <p v-if="profile.last_login">Last login: {{ formatDate(profile.last_login) }}</p>
          <p>Member since: {{ formatDate(profile.created_at) }}</p>
        </div>
      </div>

      <!-- Profile details -->
      <div class="card space-y-4">
        <h3 class="font-semibold text-gray-900 flex items-center gap-2">
          <UserCircleIcon class="w-5 h-5 text-blue-600" />
          Profile Details
        </h3>

        <div>
          <label class="form-label">Username</label>
          <input :value="profile.username" class="form-input bg-gray-50" disabled />
          <p class="text-xs text-gray-400 mt-1">Username cannot be changed. Contact a super admin if needed.</p>
        </div>

        <div>
          <label class="form-label">Full Name <span class="text-red-500">*</span></label>
          <input v-model="profileForm.full_name" class="form-input" placeholder="Your full name" />
        </div>

        <div>
          <label class="form-label">Email <span class="text-red-500">*</span></label>
          <input v-model="profileForm.email" type="email" class="form-input" placeholder="you@example.com" />
        </div>

        <div class="flex justify-end">
          <button @click="saveProfile" class="btn-primary" :disabled="savingProfile">
            <span v-if="savingProfile" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
            <CheckIcon v-else class="w-4 h-4" />
            Save Profile
          </button>
        </div>
      </div>

      <!-- Change password -->
      <div class="card space-y-4">
        <h3 class="font-semibold text-gray-900 flex items-center gap-2">
          <LockClosedIcon class="w-5 h-5 text-blue-600" />
          Change Password
        </h3>

        <div>
          <label class="form-label">Current Password <span class="text-red-500">*</span></label>
          <input v-model="passwordForm.current_password" type="password" class="form-input" autocomplete="current-password" />
        </div>

        <div>
          <label class="form-label">New Password <span class="text-red-500">*</span></label>
          <input v-model="passwordForm.new_password" type="password" class="form-input" autocomplete="new-password" />
        </div>

        <div>
          <label class="form-label">Confirm New Password <span class="text-red-500">*</span></label>
          <input v-model="passwordForm.confirm_password" type="password" class="form-input" autocomplete="new-password" />
        </div>

        <ul class="text-xs text-gray-500 space-y-1 bg-gray-50 rounded-lg p-3">
          <li :class="ruleClass(passwordRules.minLength)">At least 12 characters</li>
          <li :class="ruleClass(passwordRules.upper)">One uppercase letter</li>
          <li :class="ruleClass(passwordRules.lower)">One lowercase letter</li>
          <li :class="ruleClass(passwordRules.digit)">One digit</li>
          <li :class="ruleClass(passwordRules.special)">One special character</li>
        </ul>

        <div class="flex justify-end">
          <button @click="changePassword" class="btn-primary" :disabled="savingPassword">
            <span v-if="savingPassword" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
            <KeyIcon v-else class="w-4 h-4" />
            Update Password
          </button>
        </div>
      </div>

      <!-- MFA status (read-only info) -->
      <div v-if="profile.mfa_enabled" class="card flex items-center gap-3 text-sm text-green-700 bg-green-50 border border-green-100">
        <ShieldCheckIcon class="w-5 h-5 flex-shrink-0" />
        Two-factor authentication is enabled on your account.
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'vue-toastification'
import {
  UserCircleIcon, LockClosedIcon, CheckIcon, KeyIcon, ShieldCheckIcon,
} from '@heroicons/vue/24/outline'
import { authAPI } from '@/api'
import { useAuthStore } from '@/store/auth'

const toast = useToast()
const router = useRouter()
const authStore = useAuthStore()

const loading = ref(true)
const savingProfile = ref(false)
const savingPassword = ref(false)
const profile = ref({})

const profileForm = ref({ full_name: '', email: '' })
const passwordForm = ref({
  current_password: '',
  new_password: '',
  confirm_password: '',
})

const initials = computed(() => {
  const name = profile.value.full_name || profile.value.username || '?'
  return name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
})

const passwordRules = computed(() => {
  const pw = passwordForm.value.new_password
  return {
    minLength: pw.length >= 12,
    upper: /[A-Z]/.test(pw),
    lower: /[a-z]/.test(pw),
    digit: /\d/.test(pw),
    special: /[^A-Za-z0-9]/.test(pw),
  }
})

function ruleClass(ok) {
  return ok ? 'text-green-600' : 'text-gray-400'
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString()
}

async function loadProfile() {
  loading.value = true
  try {
    const { data } = await authAPI.getProfile()
    profile.value = data
    profileForm.value = {
      full_name: data.full_name || '',
      email: data.email || '',
    }
  } catch {
    toast.error('Failed to load profile')
  }
  loading.value = false
}

async function saveProfile() {
  if (!profileForm.value.full_name.trim()) {
    toast.error('Full name is required')
    return
  }
  if (!profileForm.value.email.trim()) {
    toast.error('Email is required')
    return
  }

  savingProfile.value = true
  try {
    const { data } = await authAPI.updateProfile(profileForm.value)
    profile.value = data.user
    authStore.setUser(data.user)
    toast.success('Profile updated')
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed to update profile')
  }
  savingProfile.value = false
}

async function changePassword() {
  const { current_password, new_password, confirm_password } = passwordForm.value

  if (!current_password || !new_password) {
    toast.error('Please fill in all password fields')
    return
  }
  if (new_password !== confirm_password) {
    toast.error('New passwords do not match')
    return
  }
  if (new_password === current_password) {
    toast.error('New password must be different from current password')
    return
  }

  savingPassword.value = true
  try {
    await authAPI.changePassword({
      current_password,
      new_password,
    })
    toast.success('Password changed. Please sign in again.')
    passwordForm.value = { current_password: '', new_password: '', confirm_password: '' }
    await authStore.logout()
    router.push('/login')
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed to change password')
  }
  savingPassword.value = false
}

onMounted(loadProfile)
</script>
