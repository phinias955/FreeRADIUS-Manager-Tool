<template>
  <div class="space-y-6">
    <div class="page-header">
      <div>
        <h1 class="page-title">Settings</h1>
        <p class="text-sm text-gray-500 mt-0.5">System configuration and security settings</p>
      </div>
      <button @click="saveSettings" class="btn-primary" :disabled="saving">
        <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
        <CheckIcon v-else class="w-4 h-4" />
        Save Changes
      </button>
    </div>

    <div v-if="loading" class="h-64 flex items-center justify-center">
      <span class="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full spinner"></span>
    </div>

    <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Password Policy -->
      <div class="card space-y-4">
        <h3 class="font-semibold text-gray-900 flex items-center gap-2">
          <LockClosedIcon class="w-5 h-5 text-blue-600" />
          Password Policy
        </h3>
        <div>
          <label class="form-label">Minimum Password Length</label>
          <input v-model="settings.password_min_length" type="number" min="8" max="32" class="form-input" />
        </div>
        <div>
          <label class="form-label">Password Expiry (days)</label>
          <select v-model="settings.password_expiry_days" class="form-select">
            <option value="30">30 days</option>
            <option value="60">60 days</option>
            <option value="90">90 days</option>
            <option value="180">180 days</option>
            <option value="365">365 days (1 year)</option>
            <option value="0">Never</option>
          </select>
        </div>
        <div>
          <label class="form-label">Max Device Limit Per User</label>
          <input v-model="settings.max_device_limit" type="number" min="1" max="50" class="form-input" />
        </div>
      </div>

      <!-- Security -->
      <div class="card space-y-4">
        <h3 class="font-semibold text-gray-900 flex items-center gap-2">
          <ShieldCheckIcon class="w-5 h-5 text-blue-600" />
          Security
        </h3>
        <div>
          <label class="form-label">Session Timeout (seconds)</label>
          <input v-model="settings.session_timeout" type="number" min="300" class="form-input" />
        </div>
        <div>
          <label class="form-label">Rate Limit (requests/minute)</label>
          <input v-model="settings.rate_limit_per_min" type="number" min="10" class="form-input" />
        </div>
        <div>
          <label class="form-label">Max Failed Login Attempts</label>
          <input v-model="settings.brute_force_attempts" type="number" min="3" max="20" class="form-input" />
        </div>
        <div>
          <label class="form-label">Lockout Duration (minutes)</label>
          <input v-model="settings.brute_force_lockout" type="number" min="1" class="form-input" />
        </div>
        <div class="flex items-center gap-2">
          <input
            v-model="mfaRequired"
            type="checkbox"
            id="mfa-required"
            class="rounded"
          />
          <label for="mfa-required" class="text-sm text-gray-700">Require MFA for all admin users</label>
        </div>
      </div>

      <!-- Email / SMTP -->
      <div class="card space-y-4">
        <h3 class="font-semibold text-gray-900 flex items-center gap-2">
          <EnvelopeIcon class="w-5 h-5 text-blue-600" />
          Email (SMTP)
        </h3>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="form-label">SMTP Host</label>
            <input v-model="settings.smtp_host" type="text" class="form-input" placeholder="smtp.gmail.com" />
          </div>
          <div>
            <label class="form-label">SMTP Port</label>
            <input v-model="settings.smtp_port" type="number" class="form-input" placeholder="587" />
          </div>
        </div>
        <div>
          <label class="form-label">SMTP Username</label>
          <input v-model="settings.smtp_user" type="text" class="form-input" placeholder="user@example.com" />
        </div>
        <div>
          <label class="form-label">From Address</label>
          <input v-model="settings.smtp_from" type="email" class="form-input" />
        </div>
        <button @click="testEmail" class="btn-secondary w-full justify-center">
          <PaperAirplaneIcon class="w-4 h-4" />
          Send Test Email
        </button>
      </div>

      <!-- Backup -->
      <div class="card space-y-4">
        <h3 class="font-semibold text-gray-900 flex items-center gap-2">
          <ArchiveBoxIcon class="w-5 h-5 text-blue-600" />
          Backup
        </h3>
        <div>
          <label class="form-label">Backup Schedule (Cron)</label>
          <input v-model="settings.backup_schedule" type="text" class="form-input" placeholder="0 2 * * *" />
          <p class="text-xs text-gray-500 mt-1">Current: {{ crondesc }}</p>
        </div>
        <div>
          <label class="form-label">Retention (days)</label>
          <input v-model="settings.backup_retention" type="number" min="1" class="form-input" />
        </div>
        <div class="flex gap-2">
          <button @click="createBackup" class="btn-secondary flex-1 justify-center">
            <ArrowDownTrayIcon class="w-4 h-4" />
            Backup Now
          </button>
        </div>
      </div>

      <!-- RADIUS Test -->
      <div class="card space-y-4 lg:col-span-2">
        <h3 class="font-semibold text-gray-900 flex items-center gap-2">
          <SignalIcon class="w-5 h-5 text-blue-600" />
          RADIUS Test
        </h3>
        <p class="text-sm text-gray-600">Test authentication against the RADIUS server with a specific username/password.</p>
        <div class="grid grid-cols-3 gap-3">
          <div>
            <label class="form-label">Username</label>
            <input v-model="radiusTest.username" type="text" class="form-input" />
          </div>
          <div>
            <label class="form-label">Password</label>
            <input v-model="radiusTest.password" type="password" class="form-input" />
          </div>
          <div>
            <label class="form-label">NAS (optional)</label>
            <input v-model="radiusTest.nasname" type="text" class="form-input" placeholder="Leave blank for default" />
          </div>
        </div>
        <button @click="runRadiusTest" class="btn-primary" :disabled="testingRadius">
          <span v-if="testingRadius" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
          Run Test
        </button>
        <div v-if="radiusTestResult" class="p-4 rounded-lg border"
          :class="radiusTestResult.success ? 'bg-green-50 border-green-200 text-green-800' : 'bg-red-50 border-red-200 text-red-800'"
        >
          <p class="font-semibold">{{ radiusTestResult.success ? '✓ Authentication Successful' : '✗ Authentication Failed' }}</p>
          <p class="text-sm mt-1">{{ radiusTestResult.message }}</p>
          <p class="text-xs mt-1 opacity-75">Latency: {{ radiusTestResult.latency_ms?.toFixed(2) }}ms | Server: {{ radiusTestResult.nasname }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { settingsAPI, radiusAPI } from '@/api'
import {
  CheckIcon, LockClosedIcon, ShieldCheckIcon, EnvelopeIcon,
  ArchiveBoxIcon, SignalIcon, ArrowDownTrayIcon, PaperAirplaneIcon,
} from '@heroicons/vue/24/outline'

const toast = useToast()
const loading = ref(false)
const saving = ref(false)
const testingRadius = ref(false)
const radiusTestResult = ref(null)
const settingsRaw = ref({})

const radiusTest = ref({ username: '', password: '', nasname: '' })

const settings = ref({
  password_min_length: '12',
  password_expiry_days: '90',
  session_timeout: '3600',
  mfa_required: 'false',
  max_device_limit: '20',
  rate_limit_per_min: '100',
  brute_force_attempts: '5',
  brute_force_lockout: '15',
  smtp_host: '',
  smtp_port: '587',
  smtp_user: '',
  smtp_from: '',
  backup_schedule: '0 2 * * *',
  backup_retention: '30',
})

const mfaRequired = computed({
  get: () => settings.value.mfa_required === 'true',
  set: (v) => { settings.value.mfa_required = v ? 'true' : 'false' },
})

const crondesc = computed(() => {
  const c = settings.value.backup_schedule
  if (c === '0 2 * * *') return 'Daily at 2:00 AM'
  if (c === '0 * * * *') return 'Hourly'
  return 'Custom schedule'
})

async function loadSettings() {
  loading.value = true
  try {
    const { data } = await settingsAPI.get()
    Object.entries(data).forEach(([k, v]) => {
      if (settings.value[k] !== undefined) {
        settings.value[k] = v.value
      }
    })
  } catch { toast.error('Failed to load settings') }
  finally { loading.value = false }
}

async function saveSettings() {
  saving.value = true
  try {
    const updates = {}
    Object.entries(settings.value).forEach(([k, v]) => { updates[k] = String(v) })
    await settingsAPI.update(updates)
    toast.success('Settings saved successfully')
  } catch { toast.error('Failed to save settings') }
  finally { saving.value = false }
}

async function createBackup() {
  try {
    const { data } = await settingsAPI.createBackup()
    toast.success(data.message)
  } catch { toast.error('Backup failed') }
}

async function testEmail() {
  toast.info('Email test functionality requires SMTP configuration')
}

async function runRadiusTest() {
  testingRadius.value = true
  radiusTestResult.value = null
  try {
    const { data } = await radiusAPI.test(radiusTest.value)
    radiusTestResult.value = data
  } catch (err) {
    radiusTestResult.value = { success: false, message: err.response?.data?.error || 'Test failed', latency_ms: 0 }
  } finally {
    testingRadius.value = false
  }
}

onMounted(loadSettings)
</script>
