<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Bulk Operations</h1>
        <p class="text-sm text-gray-500 mt-0.5">Perform mass actions on multiple RADIUS users at once</p>
      </div>
    </div>

    <!-- Info -->
    <div class="card bg-amber-50 border border-amber-200 p-4 flex gap-3">
      <ExclamationTriangleIcon class="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
      <p class="text-sm text-amber-700">Bulk operations are irreversible. Double-check your user list and action before proceeding. All operations are logged.</p>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
      <!-- Operation panel -->
      <div class="card space-y-4">
        <h3 class="font-semibold text-gray-900">New Bulk Operation</h3>

        <!-- Action selector -->
        <div>
          <label class="form-label">Action <span class="text-red-500">*</span></label>
          <div class="grid grid-cols-2 gap-2">
            <button v-for="action in actions" :key="action.id"
              type="button"
              @click="selectedAction = action.id; params = {}"
              class="flex items-center gap-2 p-3 rounded-xl border text-sm transition-all"
              :class="selectedAction === action.id
                ? 'border-blue-500 bg-blue-50 text-blue-700'
                : 'border-gray-200 text-gray-600 hover:bg-gray-50'">
              <component :is="action.icon" class="w-4 h-4 flex-shrink-0" />
              {{ action.label }}
            </button>
          </div>
        </div>

        <!-- Action params -->
        <div v-if="selectedAction === 'change_plan'">
          <label class="form-label">New Plan ID</label>
          <input v-model.number="params.plan_id" type="number" class="form-input" placeholder="Enter plan ID" />
        </div>
        <div v-if="selectedAction === 'set_expiry'">
          <label class="form-label">New Expiry Date</label>
          <input v-model="params.expiry" type="date" class="form-input" />
        </div>
        <div v-if="selectedAction === 'apply_template'">
          <label class="form-label">Template ID</label>
          <input v-model.number="params.template_id" type="number" class="form-input" placeholder="Template ID" />
          <p class="text-xs text-gray-400 mt-1">Go to <router-link to="/templates" class="text-blue-500 hover:underline">RADIUS Templates</router-link> to find the ID.</p>
        </div>

        <!-- Username input -->
        <div>
          <label class="form-label">
            Usernames
            <span class="text-gray-400 font-normal">(one per line, or paste comma-separated)</span>
          </label>
          <textarea v-model="usernamesRaw" rows="8" class="form-input font-mono text-sm"
            placeholder="user1&#10;user2&#10;user3"></textarea>
          <div class="flex items-center justify-between mt-1">
            <p class="text-xs text-gray-400">{{ usernameList.length }} user{{ usernameList.length !== 1 ? 's' : '' }} entered</p>
            <button type="button" @click="usernamesRaw = ''" class="text-xs text-gray-400 hover:text-red-500">Clear</button>
          </div>
        </div>

        <button @click="runBulk" class="btn-primary w-full" :disabled="running || !selectedAction || !usernameList.length">
          <span v-if="running" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
          <BoltIcon v-else class="w-4 h-4" />
          {{ running ? 'Processing…' : `Execute on ${usernameList.length} user${usernameList.length !== 1 ? 's' : ''}` }}
        </button>
      </div>

      <!-- Results panel -->
      <div class="card space-y-3">
        <h3 class="font-semibold text-gray-900">Results</h3>

        <div v-if="!results" class="text-center py-16 text-gray-400 text-sm">
          Run an operation to see results here
        </div>

        <template v-else>
          <div class="grid grid-cols-3 gap-3">
            <div class="bg-blue-50 rounded-xl p-3 text-center">
              <p class="text-xl font-bold text-blue-700">{{ results.total }}</p>
              <p class="text-xs text-gray-500">Total</p>
            </div>
            <div class="bg-green-50 rounded-xl p-3 text-center">
              <p class="text-xl font-bold text-green-700">{{ results.success_count }}</p>
              <p class="text-xs text-gray-500">Success</p>
            </div>
            <div class="bg-red-50 rounded-xl p-3 text-center">
              <p class="text-xl font-bold text-red-600">{{ results.fail_count }}</p>
              <p class="text-xs text-gray-500">Failed</p>
            </div>
          </div>

          <div class="max-h-64 overflow-y-auto space-y-1">
            <div v-for="r in results.results" :key="r.username"
              class="flex items-center justify-between p-2 rounded-lg text-sm"
              :class="r.success ? 'bg-green-50' : 'bg-red-50'">
              <span class="font-mono font-medium" :class="r.success ? 'text-green-700' : 'text-red-700'">
                {{ r.username }}
              </span>
              <span class="text-xs" :class="r.success ? 'text-green-600' : 'text-red-500'">
                {{ r.message }}
              </span>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- History -->
    <div class="card">
      <div class="flex items-center justify-between mb-3">
        <h3 class="font-semibold text-gray-900">Operation History</h3>
        <button @click="loadHistory" class="btn-secondary text-xs py-1.5">
          <ArrowPathIcon class="w-3.5 h-3.5" /> Refresh
        </button>
      </div>
      <table class="table text-sm">
        <thead>
          <tr>
            <th>Action</th>
            <th>Total</th>
            <th>Success</th>
            <th>Failed</th>
            <th>By</th>
            <th>Time</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!history.length"><td colspan="6" class="text-center py-6 text-gray-400">No history yet</td></tr>
          <tr v-for="h in history" :key="h.id">
            <td><span class="font-mono text-xs bg-gray-100 px-2 py-0.5 rounded">{{ h.operation }}</span></td>
            <td class="font-semibold">{{ h.target_count }}</td>
            <td class="text-green-600 font-semibold">{{ h.success_count }}</td>
            <td :class="h.fail_count > 0 ? 'text-red-500 font-semibold' : 'text-gray-400'">{{ h.fail_count }}</td>
            <td class="text-xs text-gray-500">{{ h.performed_by || '—' }}</td>
            <td class="text-xs text-gray-400">{{ h.created_at?.slice(0,16) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { bulkAPI } from '@/api'
import { BoltIcon, ArrowPathIcon, ExclamationTriangleIcon, UserMinusIcon, UserPlusIcon, TrashIcon, ClipboardDocumentCheckIcon, CalendarIcon, CodeBracketSquareIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const running = ref(false)
const selectedAction = ref('')
const usernamesRaw = ref('')
const params = reactive({})
const results = ref(null)
const history = ref([])

const actions = [
  { id: 'suspend', label: 'Suspend', icon: UserMinusIcon },
  { id: 'activate', label: 'Activate', icon: UserPlusIcon },
  { id: 'delete', label: 'Delete', icon: TrashIcon },
  { id: 'change_plan', label: 'Change Plan', icon: ClipboardDocumentCheckIcon },
  { id: 'set_expiry', label: 'Set Expiry', icon: CalendarIcon },
  { id: 'apply_template', label: 'Apply Template', icon: CodeBracketSquareIcon },
  { id: 'reset_attributes', label: 'Reset Attrs', icon: ArrowPathIcon },
]

const usernameList = computed(() => {
  const raw = usernamesRaw.value.replace(/,/g, '\n')
  return [...new Set(raw.split('\n').map(u => u.trim()).filter(Boolean))]
})

async function runBulk() {
  if (!selectedAction.value) { toast.error('Select an action'); return }
  if (!usernameList.value.length) { toast.error('Enter at least one username'); return }
  if (selectedAction.value === 'delete' && !confirm(`Really DELETE ${usernameList.value.length} users? This cannot be undone.`)) return

  running.value = true
  try {
    const { data } = await bulkAPI.execute({
      usernames: usernameList.value,
      action: selectedAction.value,
      params: { ...params }
    })
    results.value = data
    toast.success(`Done: ${data.success_count} success, ${data.fail_count} failed`)
    loadHistory()
  } catch (err) {
    toast.error(err.response?.data?.error || 'Operation failed')
  } finally { running.value = false }
}

async function loadHistory() {
  try { const { data } = await bulkAPI.history(); history.value = data.data || [] }
  catch { }
}

onMounted(loadHistory)
</script>
