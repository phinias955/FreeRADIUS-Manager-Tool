<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Captive Portals</h1>
        <p class="text-sm text-gray-500 mt-0.5">Build and manage hotspot login pages</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
        <PlusIcon class="w-4 h-4" />
        New Portal
      </button>
    </div>

    <!-- Portal cards -->
    <div v-if="!portals.length && !loading" class="card text-center py-16 text-gray-400">
      <GlobeAltIcon class="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p class="font-medium">No captive portals configured</p>
      <p class="text-sm mt-1">Create a portal to provide a branded WiFi login page for each zone</p>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div v-for="p in portals" :key="p.id" class="card" :class="!p.is_active ? 'opacity-60' : ''">
        <div class="flex items-start justify-between mb-3">
          <div>
            <h3 class="font-semibold text-gray-900">{{ p.name }}</h3>
            <p class="text-xs text-gray-500 mt-0.5">Zone: {{ p.zone_name || 'No zone' }}</p>
          </div>
          <div class="flex items-center gap-2">
            <span class="badge" :class="p.is_active ? 'badge-green' : 'badge-gray'">
              {{ p.is_active ? 'Active' : 'Off' }}
            </span>
            <div class="w-6 h-6 rounded-full border-2" :style="{ background: p.primary_color }"></div>
          </div>
        </div>

        <!-- Mini preview -->
        <div class="rounded-xl overflow-hidden border border-gray-200 mb-3"
          :style="{ background: p.bg_color }">
          <div class="p-4 text-center">
            <p class="font-bold text-sm" :style="{ color: '#0f172a' }">{{ p.title }}</p>
            <p class="text-xs text-gray-500 mt-1">{{ p.subtitle || '' }}</p>
            <div class="mt-3 mx-auto max-w-[160px]">
              <div class="rounded-lg h-7 text-xs flex items-center justify-center text-white font-medium"
                :style="{ background: p.primary_color }">
                Connect to Internet
              </div>
            </div>
          </div>
        </div>

        <div class="text-xs text-gray-500 mb-3">
          Auth: <span class="font-medium text-gray-700">{{ p.auth_type }}</span>
          · Redirect: <span class="font-mono truncate max-w-[150px] inline-block align-bottom">{{ p.redirect_url }}</span>
        </div>

        <div class="flex gap-2 pt-2 border-t border-gray-100" @click.stop>
          <a :href="`/api/v1/captive/serve/${p.id}`" target="_blank"
            class="flex-1 btn-secondary text-xs py-1.5 flex items-center justify-center gap-1">
            <ArrowTopRightOnSquareIcon class="w-3.5 h-3.5" />
            Preview
          </a>
          <button @click="openEdit(p)" class="flex-1 btn-secondary text-xs py-1.5">Edit</button>
          <button @click="removePortal(p)" class="p-1.5 rounded-lg border border-red-200 text-red-500 hover:bg-red-50">
            <TrashIcon class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal with live preview -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-4xl max-h-[95vh] overflow-hidden flex flex-col">
        <div class="flex items-center justify-between p-5 border-b flex-shrink-0">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Captive Portal</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>

        <div class="flex flex-1 overflow-hidden">
          <!-- Form -->
          <div class="w-1/2 overflow-y-auto p-5 border-r space-y-3">
            <div class="grid grid-cols-2 gap-3">
              <div class="col-span-2">
                <label class="form-label">Portal Name <span class="text-red-500">*</span></label>
                <input v-model="form.name" type="text" class="form-input" required placeholder="e.g. Mall WiFi Portal" />
              </div>
              <div class="col-span-2">
                <label class="form-label">Page Title</label>
                <input v-model="form.title" type="text" class="form-input" placeholder="WiFi Login" />
              </div>
              <div class="col-span-2">
                <label class="form-label">Subtitle</label>
                <input v-model="form.subtitle" type="text" class="form-input" placeholder="Welcome! Please sign in to continue." />
              </div>
              <div>
                <label class="form-label">Background Color</label>
                <div class="flex gap-2">
                  <input type="color" v-model="form.bg_color" class="h-9 w-12 rounded cursor-pointer border border-gray-200" />
                  <input v-model="form.bg_color" type="text" class="form-input flex-1" placeholder="#f0f4ff" />
                </div>
              </div>
              <div>
                <label class="form-label">Primary Color</label>
                <div class="flex gap-2">
                  <input type="color" v-model="form.primary_color" class="h-9 w-12 rounded cursor-pointer border border-gray-200" />
                  <input v-model="form.primary_color" type="text" class="form-input flex-1" placeholder="#3b82f6" />
                </div>
              </div>
              <div class="col-span-2">
                <label class="form-label">Auth Type</label>
                <select v-model="form.auth_type" class="form-input">
                  <option value="userpass">Username + Password</option>
                  <option value="voucher">Voucher Code</option>
                  <option value="both">Both (tabs)</option>
                </select>
              </div>
              <div class="col-span-2">
                <label class="form-label">Redirect URL (after login)</label>
                <input v-model="form.redirect_url" type="url" class="form-input" placeholder="http://example.com" />
              </div>
              <div class="col-span-2">
                <label class="form-label">Logo URL</label>
                <input v-model="form.logo_url" type="url" class="form-input" placeholder="https://..." />
              </div>
              <div class="col-span-2">
                <label class="form-label">Terms Text</label>
                <input v-model="form.terms_text" type="text" class="form-input" placeholder="By connecting you agree to our terms." />
              </div>
              <div class="col-span-2">
                <label class="form-label">Footer Text</label>
                <input v-model="form.footer_text" type="text" class="form-input" placeholder="Powered by RADIUS Manager" />
              </div>
              <div class="col-span-2 flex items-center gap-2">
                <input type="checkbox" id="cp_active" v-model="form.is_active" class="rounded" />
                <label for="cp_active" class="text-sm text-gray-700">Portal is active</label>
              </div>
            </div>
            <div class="flex justify-end gap-3 pt-2">
              <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
              <button @click="save" class="btn-primary" :disabled="saving">
                <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
                {{ editing ? 'Update' : 'Create' }}
              </button>
            </div>
          </div>

          <!-- Live preview -->
          <div class="w-1/2 overflow-y-auto" :style="{ background: form.bg_color || '#f0f4ff' }">
            <div class="flex items-center justify-center min-h-full p-8">
              <div class="bg-white rounded-2xl shadow-xl p-8 w-full max-w-[300px]">
                <div class="text-center mb-5">
                  <img v-if="form.logo_url" :src="form.logo_url" alt="logo"
                    class="max-h-16 mx-auto mb-3 rounded-lg" />
                  <h2 class="font-bold text-gray-900" style="font-size:18px">{{ form.title || 'WiFi Login' }}</h2>
                  <p v-if="form.subtitle" class="text-sm text-gray-500 mt-1">{{ form.subtitle }}</p>
                </div>
                <div class="space-y-2 mb-4">
                  <template v-if="form.auth_type !== 'voucher'">
                    <div class="h-9 rounded-lg border border-gray-200 bg-gray-50 flex items-center px-3 text-xs text-gray-400">Username</div>
                    <div class="h-9 rounded-lg border border-gray-200 bg-gray-50 flex items-center px-3 text-xs text-gray-400">Password</div>
                  </template>
                  <template v-if="form.auth_type !== 'userpass'">
                    <div class="h-9 rounded-lg border border-gray-200 bg-gray-50 flex items-center px-3 text-xs text-gray-400">Voucher code</div>
                  </template>
                  <div class="h-10 rounded-xl flex items-center justify-center text-white text-sm font-semibold"
                    :style="{ background: form.primary_color || '#3b82f6' }">
                    Connect to Internet
                  </div>
                </div>
                <p v-if="form.terms_text" class="text-xs text-gray-400 text-center">{{ form.terms_text }}</p>
              </div>
            </div>
            <p class="text-center text-xs text-gray-400 pb-4">{{ form.footer_text || 'Powered by RADIUS Manager' }}</p>
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
import { captiveAPI } from '@/api'
import { PlusIcon, XMarkIcon, TrashIcon, GlobeAltIcon, ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const editing = ref(null)
const portals = ref([])
const form = reactive({ name: '', title: 'WiFi Login', subtitle: '', logo_url: '', bg_color: '#f0f4ff', primary_color: '#3b82f6', auth_type: 'userpass', redirect_url: 'http://example.com', terms_text: '', footer_text: '', is_active: true, zone_id: null })

async function load() {
  loading.value = true
  try {
    const { data } = await captiveAPI.list()
    portals.value = data.data || []
  } catch { } finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', title: 'WiFi Login', subtitle: '', logo_url: '', bg_color: '#f0f4ff', primary_color: '#3b82f6', auth_type: 'userpass', redirect_url: 'http://example.com', terms_text: '', footer_text: '', is_active: true, zone_id: null })
  showModal.value = true
}

function openEdit(p) {
  editing.value = p.id
  Object.assign(form, { name: p.name, title: p.title, subtitle: p.subtitle || '', logo_url: p.logo_url || '', bg_color: p.bg_color, primary_color: p.primary_color, auth_type: p.auth_type, redirect_url: p.redirect_url, terms_text: p.terms_text || '', footer_text: p.footer_text || '', is_active: p.is_active, zone_id: p.zone_id })
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) { await captiveAPI.update(editing.value, form); toast.success('Updated') }
    else { await captiveAPI.create(form); toast.success('Created') }
    showModal.value = false; load()
  } catch (err) { toast.error(err.response?.data?.error || 'Failed') }
  finally { saving.value = false }
}

async function removePortal(p) {
  if (!confirm(`Delete portal "${p.name}"?`)) return
  try { await captiveAPI.delete(p.id); toast.success('Deleted'); load() }
  catch { toast.error('Failed') }
}

onMounted(load)
</script>
