<template>
  <div class="p-6 space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <BeakerIcon class="w-7 h-7 text-blue-500" />
          RADIUS Simulator
        </h1>
        <p class="text-sm text-gray-500 mt-1">Send real UDP Access-Request packets and inspect the full RADIUS exchange</p>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-5 gap-6">
      <!-- ── Request form ─────────────────────────────────────────────── -->
      <div class="lg:col-span-2 space-y-4">
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-5 space-y-4">
          <h2 class="font-semibold text-gray-900 dark:text-white">Single Auth Test</h2>

          <div class="space-y-3">
            <div>
              <label class="label">Username *</label>
              <input v-model="form.username" class="input w-full" placeholder="testuser" />
            </div>
            <div>
              <label class="label">Password *</label>
              <input v-model="form.password" type="password" class="input w-full" placeholder="••••••••" />
            </div>
            <div class="grid grid-cols-2 gap-2">
              <div>
                <label class="label">NAS IP</label>
                <input v-model="form.nas_ip" class="input w-full" placeholder="freeradius" />
              </div>
              <div>
                <label class="label">Port</label>
                <input v-model.number="form.nas_port" class="input w-full" type="number" placeholder="1812" />
              </div>
            </div>
            <div>
              <label class="label">Shared Secret</label>
              <input v-model="form.secret" class="input w-full" placeholder="testing123 (from env if empty)" />
            </div>
            <div>
              <label class="label">Called-Station-Id</label>
              <input v-model="form.called_station_id" class="input w-full" placeholder="optional" />
            </div>
            <div>
              <label class="label">Calling-Station-Id</label>
              <input v-model="form.calling_station_id" class="input w-full" placeholder="optional" />
            </div>
            <div>
              <label class="label">Timeout (ms)</label>
              <input v-model.number="form.timeout_ms" class="input w-full" type="number" placeholder="5000" />
            </div>
          </div>

          <button @click="runTest" :disabled="testing || !form.username || !form.password"
            class="w-full btn-primary flex items-center justify-center gap-2">
            <PlayIcon class="w-4 h-4" />
            {{ testing ? 'Sending…' : 'Send Access-Request' }}
          </button>
        </div>

        <!-- ── Batch test ───────────────────────────────────────────────── -->
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-5 space-y-3">
          <h2 class="font-semibold text-gray-900 dark:text-white">Batch Test</h2>
          <p class="text-xs text-gray-500">Paste CSV: username,password (one per line, max 20)</p>
          <textarea v-model="batchCSV" rows="5" class="input w-full font-mono text-xs"
            placeholder="alice,pass1&#10;bob,pass2" />
          <button @click="runBatch" :disabled="batchTesting || !batchCSV.trim()" class="w-full btn-secondary">
            {{ batchTesting ? 'Testing…' : 'Run Batch' }}
          </button>
          <div v-if="batchResult" class="grid grid-cols-3 gap-2 text-center">
            <div class="bg-gray-50 dark:bg-gray-700 rounded p-2">
              <div class="font-bold text-gray-800 dark:text-gray-200">{{ batchResult.total }}</div>
              <div class="text-xs text-gray-500">Total</div>
            </div>
            <div class="bg-green-50 rounded p-2">
              <div class="font-bold text-green-600">{{ batchResult.accepted }}</div>
              <div class="text-xs text-green-500">Accepted</div>
            </div>
            <div class="bg-red-50 rounded p-2">
              <div class="font-bold text-red-600">{{ batchResult.rejected }}</div>
              <div class="text-xs text-red-500">Rejected</div>
            </div>
          </div>
          <div v-if="batchResult?.results" class="space-y-1 max-h-48 overflow-y-auto">
            <div v-for="(r, i) in batchResult.results" :key="i"
              class="flex items-center gap-2 text-xs p-1.5 rounded"
              :class="r.reply === 'Access-Accept' ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20'">
              <CheckCircleIcon v-if="r.reply === 'Access-Accept'" class="w-4 h-4 text-green-500 flex-shrink-0" />
              <XCircleIcon v-else class="w-4 h-4 text-red-500 flex-shrink-0" />
              <span class="font-mono">{{ r.request_attrs?.['User-Name'] }}</span>
              <span class="ml-auto font-medium" :class="r.reply === 'Access-Accept' ? 'text-green-600' : 'text-red-600'">
                {{ r.reply }}
              </span>
              <span class="text-gray-400">{{ r.latency_ms }}ms</span>
            </div>
          </div>
        </div>
      </div>

      <!-- ── Result panel ────────────────────────────────────────────────── -->
      <div class="lg:col-span-3 space-y-4">
        <!-- Empty state -->
        <div v-if="!result" class="bg-white dark:bg-gray-800 rounded-xl shadow p-10 flex flex-col items-center justify-center gap-3 text-gray-400 h-full">
          <BeakerIcon class="w-12 h-12" />
          <p class="text-sm">Run a test to see the RADIUS exchange</p>
        </div>

        <template v-else>
          <!-- Result banner -->
          <div class="rounded-xl p-4 flex items-center gap-4"
            :class="result.reply === 'Access-Accept'
              ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700'
              : result.reply === 'Access-Reject'
              ? 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700'
              : 'bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200'">
            <CheckCircleIcon v-if="result.reply === 'Access-Accept'" class="w-8 h-8 text-green-500" />
            <XCircleIcon v-else-if="result.reply === 'Access-Reject'" class="w-8 h-8 text-red-500" />
            <ExclamationTriangleIcon v-else class="w-8 h-8 text-yellow-500" />
            <div>
              <div class="text-lg font-bold" :class="result.reply === 'Access-Accept' ? 'text-green-700' : result.reply === 'Access-Reject' ? 'text-red-700' : 'text-yellow-700'">
                {{ result.error || result.reply }}
              </div>
              <div class="text-sm text-gray-500">Latency: {{ result.latency_ms }}ms</div>
            </div>
          </div>

          <!-- Attributes side-by-side -->
          <div class="grid grid-cols-2 gap-4">
            <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
              <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-1">
                <ArrowUpCircleIcon class="w-4 h-4" /> Request Attributes
              </h3>
              <div class="space-y-1">
                <div v-for="(val, key) in result.request_attrs" :key="key"
                  class="text-xs font-mono bg-gray-50 dark:bg-gray-700 rounded p-1.5">
                  <span class="text-blue-600 dark:text-blue-400">{{ key }}</span>
                  <span class="text-gray-500"> = </span>
                  <span class="text-gray-900 dark:text-gray-100">{{ val }}</span>
                </div>
              </div>
            </div>
            <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4">
              <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-1">
                <ArrowDownCircleIcon class="w-4 h-4" /> Reply Attributes
              </h3>
              <div class="space-y-1">
                <div v-for="(val, key) in result.reply_attrs" :key="key"
                  class="text-xs font-mono bg-gray-50 dark:bg-gray-700 rounded p-1.5">
                  <span class="text-green-600 dark:text-green-400">{{ key }}</span>
                  <span class="text-gray-500"> = </span>
                  <span class="text-gray-900 dark:text-gray-100">{{ val }}</span>
                </div>
                <p v-if="!Object.keys(result.reply_attrs || {}).length" class="text-xs text-gray-400">
                  No reply attributes
                </p>
              </div>
            </div>
          </div>

          <!-- Raw hex -->
          <div class="bg-white dark:bg-gray-800 rounded-xl shadow p-4 space-y-3">
            <button @click="showRaw = !showRaw" class="text-sm text-blue-500 hover:underline flex items-center gap-1">
              <CodeBracketIcon class="w-4 h-4" />
              {{ showRaw ? 'Hide' : 'Show' }} Raw Packet Hex
            </button>
            <div v-if="showRaw" class="space-y-2">
              <div>
                <div class="text-xs text-gray-500 mb-1">Request (hex)</div>
                <pre class="text-xs font-mono bg-gray-50 dark:bg-gray-700 rounded p-2 overflow-x-auto break-all whitespace-pre-wrap">{{ hexFormat(result.raw_request_hex) }}</pre>
              </div>
              <div>
                <div class="text-xs text-gray-500 mb-1">Reply (hex)</div>
                <pre class="text-xs font-mono bg-gray-50 dark:bg-gray-700 rounded p-2 overflow-x-auto break-all whitespace-pre-wrap">{{ hexFormat(result.raw_reply_hex) }}</pre>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import {
  BeakerIcon, PlayIcon, CheckCircleIcon, XCircleIcon,
  ExclamationTriangleIcon, ArrowUpCircleIcon, ArrowDownCircleIcon,
  CodeBracketIcon
} from '@heroicons/vue/24/outline'
import { useToast } from 'vue-toastification'
import { securityAPI } from '../api'

const toast = useToast()
const testing = ref(false)
const batchTesting = ref(false)
const showRaw = ref(false)
const result = ref(null)
const batchResult = ref(null)
const batchCSV = ref('')

const form = ref({
  username: '',
  password: '',
  nas_ip: '',
  nas_port: 1812,
  secret: '',
  called_station_id: '',
  calling_station_id: '',
  timeout_ms: 5000,
})

async function runTest() {
  testing.value = true
  result.value = null
  try {
    const r = await securityAPI.simulateAuth(form.value)
    result.value = r.data
  } catch (err) {
    toast.error('Test failed: ' + (err.response?.data?.error || err.message))
  }
  testing.value = false
}

async function runBatch() {
  batchTesting.value = true
  batchResult.value = null
  try {
    const pairs = batchCSV.value.trim().split('\n')
      .map(line => {
        const [username, ...rest] = line.split(',')
        return { username: username.trim(), password: rest.join(',').trim() }
      })
      .filter(p => p.username && p.password)
      .slice(0, 20)

    const r = await securityAPI.simulateBatch({ pairs })
    batchResult.value = r.data
  } catch (err) {
    toast.error('Batch test failed: ' + (err.response?.data?.error || err.message))
  }
  batchTesting.value = false
}

function hexFormat(hex) {
  if (!hex) return '(none)'
  return hex.match(/.{1,32}/g)?.join('\n') || hex
}
</script>

<style scoped>
.label { @apply block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1; }
.input { @apply block border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500; }
.btn-primary { @apply bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed; }
.btn-secondary { @apply bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50; }
</style>
