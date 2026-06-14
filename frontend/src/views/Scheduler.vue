<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Scheduler</h1>
        <p class="text-sm text-gray-500 mt-0.5">Automated background tasks — expires accounts, marks overdue invoices, cleans sessions</p>
      </div>
    </div>

    <!-- Task cards -->
    <div v-if="loading" class="text-center py-16 text-gray-400">Loading…</div>
    <div v-else class="space-y-3">
      <div v-for="task in tasks" :key="task.id" class="card p-4 flex items-center justify-between gap-4"
        :class="!task.is_active ? 'opacity-60' : ''">
        <div class="flex items-center gap-3 flex-1 min-w-0">
          <div class="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0"
            :class="taskColor(task.task_type)">
            <component :is="taskIcon(task.task_type)" class="w-5 h-5" />
          </div>
          <div class="min-w-0">
            <p class="font-medium text-sm text-gray-900">{{ task.name }}</p>
            <p class="text-xs text-gray-500 mt-0.5">
              Schedule: <span class="font-medium capitalize">{{ task.schedule }}</span>
              <span v-if="task.next_run" class="ml-2">— next: {{ formatDate(task.next_run) }}</span>
            </p>
            <p v-if="task.last_result" class="text-xs mt-0.5 truncate"
              :class="task.last_result.includes('failed') ? 'text-red-500' : 'text-green-600'">
              Last: {{ task.last_result }}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2 flex-shrink-0">
          <!-- Schedule selector -->
          <select v-model="task.schedule" @change="updateSchedule(task)"
            class="form-input py-1 text-xs w-24">
            <option value="hourly">Hourly</option>
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
          </select>

          <!-- Status badge -->
          <span class="badge" :class="task.is_active ? 'badge-green' : 'badge-gray'">
            {{ task.is_active ? 'Active' : 'Paused' }}
          </span>

          <!-- Run Now -->
          <button @click="runNow(task)" :disabled="running === task.id"
            class="btn-secondary text-xs py-1.5 px-3">
            <span v-if="running === task.id" class="w-3 h-3 border border-gray-400 border-t-transparent rounded-full spinner"></span>
            <PlayIcon v-else class="w-3.5 h-3.5" />
            Run Now
          </button>

          <!-- Toggle -->
          <button @click="toggle(task)" class="btn-secondary text-xs py-1.5 px-3">
            {{ task.is_active ? 'Pause' : 'Enable' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Last run log -->
    <div class="card">
      <h3 class="font-semibold text-gray-900 mb-3">Recent Task Runs</h3>
      <div v-if="!runLog.length" class="text-center py-8 text-gray-400 text-sm">
        No runs recorded yet. Use "Run Now" to execute a task immediately.
      </div>
      <div v-else class="space-y-2">
        <div v-for="(log, i) in runLog" :key="i" class="flex items-start gap-3 text-sm p-3 rounded-lg"
          :class="log.type === 'success' ? 'bg-green-50' : 'bg-red-50'">
          <CheckCircleIcon v-if="log.type === 'success'" class="w-4 h-4 text-green-600 flex-shrink-0 mt-0.5" />
          <ExclamationCircleIcon v-else class="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" />
          <div>
            <p class="font-medium" :class="log.type === 'success' ? 'text-green-800' : 'text-red-700'">{{ log.task }}</p>
            <p class="text-xs mt-0.5 opacity-75">{{ log.result }} — {{ log.time }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { schedulerAPI } from '@/api'
import { format, parseISO } from 'date-fns'
import {
  PlayIcon, CheckCircleIcon, ExclamationCircleIcon,
  UserMinusIcon, DocumentCurrencyDollarIcon, ServerStackIcon,
  ClockIcon, ArrowPathIcon,
} from '@heroicons/vue/24/outline'

const toast = useToast()
const loading = ref(false)
const running = ref(null)
const tasks = ref([])
const runLog = ref([])

function formatDate(d) {
  try { return format(parseISO(d), 'MMM d, HH:mm') } catch { return d }
}

function taskColor(type) {
  return {
    expire_accounts: 'bg-red-100 text-red-600',
    mark_overdue_invoices: 'bg-orange-100 text-orange-600',
    cleanup_sessions: 'bg-blue-100 text-blue-600',
    daily_report: 'bg-purple-100 text-purple-600',
    renew_plans: 'bg-green-100 text-green-600',
  }[type] || 'bg-gray-100 text-gray-600'
}

function taskIcon(type) {
  return {
    expire_accounts: UserMinusIcon,
    mark_overdue_invoices: DocumentCurrencyDollarIcon,
    cleanup_sessions: ServerStackIcon,
    daily_report: ClockIcon,
    renew_plans: ArrowPathIcon,
  }[type] || ClockIcon
}

async function load() {
  loading.value = true
  try {
    const { data } = await schedulerAPI.list()
    tasks.value = data.data || []
  } catch { /* silent */ } finally { loading.value = false }
}

async function runNow(task) {
  running.value = task.id
  try {
    const { data } = await schedulerAPI.runNow(task.id)
    const logEntry = {
      task: task.name,
      result: `${data.message} (${data.affected} rows affected)`,
      time: format(new Date(), 'HH:mm:ss'),
      type: 'success',
    }
    runLog.value.unshift(logEntry)
    toast.success(`${task.name}: ${data.message}`)
    load()
  } catch (err) {
    runLog.value.unshift({ task: task.name, result: err.response?.data?.error || 'failed', time: format(new Date(), 'HH:mm:ss'), type: 'error' })
    toast.error('Task failed')
  } finally { running.value = null }
}

async function toggle(task) {
  try {
    await schedulerAPI.toggle(task.id)
    load()
  } catch { toast.error('Failed') }
}

async function updateSchedule(task) {
  try {
    await schedulerAPI.updateSchedule(task.id, task.schedule)
    toast.success('Schedule updated')
  } catch { toast.error('Failed') }
}

onMounted(load)
</script>
