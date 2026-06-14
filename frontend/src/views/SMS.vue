<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">SMS Notifications</h1>
        <p class="text-sm text-gray-500 mt-0.5">Send SMS alerts to users via HTTP gateway</p>
      </div>
    </div>

    <!-- Gateway status -->
    <div class="card p-4 flex items-center gap-4"
      :class="smsConfig.configured ? 'bg-green-50 border-green-200' : 'bg-yellow-50 border-yellow-200'"
      style="border-width:1px">
      <div class="w-10 h-10 rounded-xl flex items-center justify-center"
        :class="smsConfig.configured ? 'bg-green-100' : 'bg-yellow-100'">
        <DevicePhoneMobileIcon class="w-5 h-5" :class="smsConfig.configured ? 'text-green-600' : 'text-yellow-600'" />
      </div>
      <div class="flex-1">
        <p class="font-semibold" :class="smsConfig.configured ? 'text-green-900' : 'text-yellow-900'">
          SMS Gateway: {{ smsConfig.configured ? 'Configured' : 'Not configured' }}
        </p>
        <p class="text-xs mt-0.5" :class="smsConfig.configured ? 'text-green-700' : 'text-yellow-700'">
          {{ smsConfig.configured
            ? `Sender ID: ${smsConfig.sender_id || 'RADIUS'} | Gateway: ${smsConfig.gateway}`
            : 'Set SMS_GATEWAY, SMS_API_KEY, SMS_SENDER_ID in your .env file to enable SMS sending.' }}
        </p>
      </div>
    </div>

    <!-- Actions tabs -->
    <div class="card p-0 overflow-hidden">
      <div class="flex border-b border-gray-100">
        <button v-for="tab in tabs" :key="tab.id" @click="activeTab = tab.id"
          class="flex-1 py-3 text-sm font-medium transition-colors"
          :class="activeTab === tab.id ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50/50' : 'text-gray-500 hover:text-gray-700'">
          {{ tab.label }}
        </button>
      </div>

      <div class="p-6">
        <!-- Send single SMS -->
        <div v-if="activeTab === 'send'">
          <form @submit.prevent="sendSMS" class="space-y-4 max-w-md">
            <div>
              <label class="form-label">Phone Number <span class="text-red-500">*</span></label>
              <input v-model="singleMsg.to" type="tel" class="form-input" required placeholder="+263771234567" />
            </div>
            <div>
              <label class="form-label">Username (optional)</label>
              <input v-model="singleMsg.username" type="text" class="form-input" placeholder="Link to a RADIUS user" />
            </div>
            <div>
              <label class="form-label">Message <span class="text-red-500">*</span></label>
              <textarea v-model="singleMsg.message" rows="4" class="form-input" required
                placeholder="Type your message…" maxlength="320"></textarea>
              <p class="text-xs text-gray-400 mt-1">{{ singleMsg.message.length }}/320</p>
            </div>
            <button type="submit" class="btn-primary" :disabled="sending">
              <span v-if="sending" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Send SMS
            </button>
          </form>
        </div>

        <!-- Notify expiring users -->
        <div v-if="activeTab === 'expiry'">
          <p class="text-sm text-gray-600 mb-4">Send SMS to users whose accounts expire within the next N days (only users with a phone number set).</p>
          <form @submit.prevent="notifyExpiry" class="space-y-4 max-w-md">
            <div>
              <label class="form-label">Notify users expiring within (days)</label>
              <input v-model.number="expiryForm.days" type="number" min="1" max="30" class="form-input" />
            </div>
            <div>
              <label class="form-label">Message Template</label>
              <textarea v-model="expiryForm.message" rows="4" class="form-input"
                placeholder="Use {username} and {expiry} placeholders"></textarea>
              <p class="text-xs text-gray-400 mt-1">Placeholders: <code class="bg-gray-100 px-1 rounded">{username}</code>, <code class="bg-gray-100 px-1 rounded">{expiry}</code></p>
            </div>
            <button type="submit" class="btn-primary" :disabled="sendingBulk">
              <span v-if="sendingBulk" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Send Expiry Notifications
            </button>
            <div v-if="expiryResult" class="p-3 rounded-xl text-sm"
              :class="expiryResult.failed > 0 ? 'bg-yellow-50 text-yellow-700' : 'bg-green-50 text-green-700'">
              {{ expiryResult.message }}
            </div>
          </form>
        </div>

        <!-- SMS Logs -->
        <div v-if="activeTab === 'logs'">
          <div class="flex justify-between items-center mb-3">
            <h3 class="text-sm font-medium text-gray-700">Send History</h3>
            <button @click="loadLogs" class="btn-secondary text-xs py-1.5"><ArrowPathIcon class="w-3.5 h-3.5" />Refresh</button>
          </div>
          <div v-if="loadingLogs" class="py-8 text-center text-gray-400">Loading…</div>
          <table v-else class="table text-sm">
            <thead>
              <tr>
                <th>To</th>
                <th>Message</th>
                <th>Status</th>
                <th>User</th>
                <th>Time</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!logs.length">
                <td colspan="5" class="text-center py-8 text-gray-400">No SMS messages sent yet</td>
              </tr>
              <tr v-for="l in logs" :key="l.id">
                <td class="font-mono text-xs">{{ l.recipient }}</td>
                <td class="max-w-xs truncate text-xs">{{ l.message }}</td>
                <td>
                  <span class="badge" :class="l.status === 'sent' ? 'badge-green' : l.status === 'failed' ? 'badge-red' : 'badge-gray'">
                    {{ l.status }}
                  </span>
                </td>
                <td class="text-xs text-gray-500">{{ l.username || '—' }}</td>
                <td class="text-xs text-gray-400">{{ formatDate(l.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { smsAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import { DevicePhoneMobileIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const activeTab = ref('send')
const tabs = [
  { id: 'send', label: 'Send Message' },
  { id: 'expiry', label: 'Expiry Notifications' },
  { id: 'logs', label: 'SMS Logs' },
]

const smsConfig = ref({ configured: false, gateway: '', sender_id: '' })
const singleMsg = reactive({ to: '', message: '', username: '' })
const expiryForm = reactive({ days: 3, message: 'Dear {username}, your account expires on {expiry}. Please renew to continue.' })
const expiryResult = ref(null)
const logs = ref([])
const sending = ref(false)
const sendingBulk = ref(false)
const loadingLogs = ref(false)

function formatDate(d) {
  if (!d) return ''
  try { return format(parseISO(d), 'MMM d, HH:mm') } catch { return d }
}

async function loadConfig() {
  try {
    const { data } = await smsAPI.config()
    smsConfig.value = data
  } catch { /* silent */ }
}

async function sendSMS() {
  sending.value = true
  try {
    await smsAPI.send(singleMsg)
    toast.success('SMS sent successfully')
    singleMsg.to = ''
    singleMsg.message = ''
    singleMsg.username = ''
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed to send SMS')
  } finally { sending.value = false }
}

async function notifyExpiry() {
  sendingBulk.value = true
  expiryResult.value = null
  try {
    const { data } = await smsAPI.notifyExpiry(expiryForm)
    expiryResult.value = data
  } catch (err) {
    toast.error(err.response?.data?.error || 'Failed')
  } finally { sendingBulk.value = false }
}

async function loadLogs() {
  loadingLogs.value = true
  try {
    const { data } = await smsAPI.logs()
    logs.value = data.data || []
  } catch { /* silent */ } finally { loadingLogs.value = false }
}

onMounted(() => {
  loadConfig()
  loadLogs()
})
</script>
