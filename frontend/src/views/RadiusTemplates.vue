<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">RADIUS Templates</h1>
        <p class="text-sm text-gray-500 mt-0.5">Reusable attribute presets — apply to users with one click</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
        <PlusIcon class="w-4 h-4" />
        New Template
      </button>
    </div>

    <!-- Template grid -->
    <div v-if="!templates.length && !loading" class="card text-center py-16 text-gray-400">
      <CodeBracketSquareIcon class="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p class="font-medium">No templates yet</p>
      <p class="text-sm mt-1">Create templates for common RADIUS configs like bandwidth limits, session timeouts</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-for="t in templates" :key="t.id"
        class="card" :class="!t.is_active ? 'opacity-60' : ''">
        <div class="flex items-start justify-between mb-3">
          <div>
            <h3 class="font-semibold text-gray-900">{{ t.name }}</h3>
            <p v-if="t.description" class="text-xs text-gray-500 mt-0.5">{{ t.description }}</p>
          </div>
          <span class="badge" :class="t.is_active ? 'badge-green' : 'badge-gray'">
            {{ t.is_active ? 'Active' : 'Off' }}
          </span>
        </div>

        <!-- Attributes list -->
        <div class="space-y-1 mb-3">
          <div v-if="!t.attributes?.length" class="text-xs text-gray-400">No attributes defined</div>
          <div v-for="(attr, i) in t.attributes" :key="i"
            class="flex items-center gap-2 bg-gray-50 rounded-lg p-2 text-xs font-mono">
            <span class="w-2 h-2 rounded-full flex-shrink-0"
              :class="attr.table === 'check' ? 'bg-orange-400' : 'bg-blue-400'"></span>
            <span class="text-gray-700 font-semibold">{{ attr.attribute }}</span>
            <span class="text-gray-400">{{ attr.op }}</span>
            <span class="text-green-700 font-medium truncate">{{ attr.value }}</span>
          </div>
        </div>

        <div class="flex items-center gap-1 text-xs text-gray-400 mb-3">
          <span class="w-2 h-2 rounded-full bg-orange-400"></span> check (radcheck)
          <span class="w-2 h-2 rounded-full bg-blue-400 ml-2"></span> reply (radreply)
        </div>

        <div class="flex gap-2 pt-2 border-t border-gray-100" v-if="authStore.isAdmin">
          <button @click="openApply(t)" class="flex-1 btn-secondary text-xs py-1.5 text-blue-600">
            <BoltIcon class="w-3.5 h-3.5" />
            Apply to Users
          </button>
          <button @click="cloneTemplate(t)" class="btn-secondary text-xs py-1.5 px-2.5">
            <DocumentDuplicateIcon class="w-3.5 h-3.5" />
          </button>
          <button @click="openEdit(t)" class="btn-secondary text-xs py-1.5 px-2.5">
            <PencilIcon class="w-3.5 h-3.5" />
          </button>
          <button @click="removeTemplate(t)" class="p-1.5 rounded-lg border border-red-200 text-red-500 hover:bg-red-50">
            <TrashIcon class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-xl max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between p-6 border-b sticky top-0 bg-white z-10">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Template</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <div class="p-6 space-y-4">
          <div>
            <label class="form-label">Template Name <span class="text-red-500">*</span></label>
            <input v-model="form.name" type="text" class="form-input" required placeholder="e.g. Basic-5Mbps" />
          </div>
          <div>
            <label class="form-label">Description</label>
            <input v-model="form.description" type="text" class="form-input" placeholder="Brief description" />
          </div>

          <!-- Attributes editor -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <label class="form-label mb-0">Attributes</label>
              <button type="button" @click="addAttr" class="btn-secondary text-xs py-1">
                <PlusIcon class="w-3 h-3" /> Add
              </button>
            </div>
            <div class="space-y-2">
              <div v-for="(attr, i) in form.attributes" :key="i"
                class="grid grid-cols-12 gap-2 items-center">
                <select v-model="attr.table" class="col-span-2 form-input text-xs py-1.5">
                  <option value="check">check</option>
                  <option value="reply">reply</option>
                </select>
                <input v-model="attr.attribute" type="text" class="col-span-4 form-input text-xs py-1.5 font-mono"
                  placeholder="Attribute" list="attr-names" />
                <select v-model="attr.op" class="col-span-2 form-input text-xs py-1.5 font-mono">
                  <option>:=</option>
                  <option>==</option>
                  <option>+=</option>
                  <option>=</option>
                </select>
                <input v-model="attr.value" type="text" class="col-span-3 form-input text-xs py-1.5 font-mono"
                  placeholder="Value" />
                <button type="button" @click="removeAttr(i)" class="col-span-1 text-red-400 hover:text-red-600">
                  <XMarkIcon class="w-4 h-4" />
                </button>
              </div>
              <datalist id="attr-names">
                <option v-for="n in commonAttrs" :key="n" :value="n" />
              </datalist>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <input type="checkbox" id="tmpl_active" v-model="form.is_active" class="rounded" />
            <label for="tmpl_active" class="text-sm text-gray-700">Template is active</label>
          </div>

          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button @click="save" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              {{ editing ? 'Update' : 'Create' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Apply Modal -->
    <div v-if="showApply" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-sm">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">Apply: {{ applyTarget?.name }}</h3>
          <button @click="showApply = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <div class="p-6 space-y-4">
          <p class="text-sm text-gray-600">Enter one username per line. All listed users will have the template attributes applied immediately.</p>
          <div>
            <label class="form-label">Usernames <span class="text-red-500">*</span></label>
            <textarea v-model="applyUsernames" rows="6" class="form-input font-mono text-sm"
              placeholder="user1&#10;user2&#10;user3"></textarea>
          </div>
          <div v-if="applyResult" class="p-3 rounded-xl text-sm"
            :class="applyResult.fail_count > 0 ? 'bg-yellow-50 text-yellow-700' : 'bg-green-50 text-green-700'">
            Applied to {{ applyResult.applied }}, failed {{ applyResult.failed }}
          </div>
          <div class="flex justify-end gap-3">
            <button @click="showApply = false" class="btn-secondary">Close</button>
            <button @click="doApply" class="btn-primary" :disabled="applying">
              <span v-if="applying" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              Apply Template
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { useAuthStore } from '@/store/auth'
import { templatesAPI } from '@/api'
import { PlusIcon, XMarkIcon, TrashIcon, PencilIcon, BoltIcon, DocumentDuplicateIcon, CodeBracketSquareIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const applying = ref(false)
const showModal = ref(false)
const showApply = ref(false)
const editing = ref(null)
const applyTarget = ref(null)
const applyUsernames = ref('')
const applyResult = ref(null)
const templates = ref([])
const form = reactive({ name: '', description: '', attributes: [], is_active: true })

const commonAttrs = [
  'Mikrotik-Rate-Limit', 'Simultaneous-Use', 'Session-Timeout', 'Idle-Timeout',
  'Max-Octets', 'Framed-IP-Address', 'Framed-Pool', 'WISPr-Bandwidth-Max-Up',
  'WISPr-Bandwidth-Max-Down', 'Auth-Type', 'Expiration', 'Service-Type'
]

function addAttr() { form.attributes.push({ attribute: '', op: ':=', value: '', table: 'reply' }) }
function removeAttr(i) { form.attributes.splice(i, 1) }

async function load() {
  loading.value = true
  try { const { data } = await templatesAPI.list(); templates.value = data.data || [] }
  catch { } finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', description: '', attributes: [], is_active: true })
  showModal.value = true
}

function openEdit(t) {
  editing.value = t.id
  Object.assign(form, { name: t.name, description: t.description || '', attributes: JSON.parse(JSON.stringify(t.attributes || [])), is_active: t.is_active })
  showModal.value = true
}

function openApply(t) {
  applyTarget.value = t
  applyUsernames.value = ''
  applyResult.value = null
  showApply.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) { await templatesAPI.update(editing.value, form); toast.success('Updated') }
    else { await templatesAPI.create(form); toast.success('Created') }
    showModal.value = false; load()
  } catch (err) { toast.error(err.response?.data?.error || 'Failed') }
  finally { saving.value = false }
}

async function doApply() {
  applying.value = true
  const usernames = applyUsernames.value.split('\n').map(u => u.trim()).filter(Boolean)
  if (!usernames.length) { toast.error('Enter at least one username'); applying.value = false; return }
  try {
    const { data } = await templatesAPI.apply(applyTarget.value.id, { usernames })
    applyResult.value = data
    toast.success(`Applied to ${data.applied} users`)
  } catch (err) { toast.error(err.response?.data?.error || 'Failed') }
  finally { applying.value = false }
}

async function cloneTemplate(t) {
  const name = prompt('Name for cloned template:', t.name + ' (copy)')
  if (!name) return
  try { await templatesAPI.clone(t.id, { name }); toast.success('Cloned'); load() }
  catch (err) { toast.error(err.response?.data?.error || 'Failed') }
}

async function removeTemplate(t) {
  if (!confirm(`Delete template "${t.name}"?`)) return
  try { await templatesAPI.delete(t.id); toast.success('Deleted'); load() }
  catch { toast.error('Failed') }
}

onMounted(load)
</script>
