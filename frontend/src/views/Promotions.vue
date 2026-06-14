<template>
  <div class="space-y-5">
    <div class="page-header">
      <div>
        <h1 class="page-title">Promotions</h1>
        <p class="text-sm text-gray-500 mt-0.5">Manage discount codes and special offers</p>
      </div>
      <button @click="openCreate" class="btn-primary" v-if="authStore.isAdmin">
        <PlusIcon class="w-4 h-4" />
        New Promo Code
      </button>
    </div>

    <!-- Validate bar -->
    <div class="card p-4">
      <p class="text-sm font-medium text-gray-700 mb-2">Test a promo code</p>
      <div class="flex gap-3">
        <input v-model="testCode" type="text" class="form-input flex-1 font-mono uppercase text-sm"
          placeholder="SUMMER20" @keyup.enter="validateCode" />
        <input v-model.number="testPrice" type="number" step="0.01" class="form-input w-28 text-sm"
          placeholder="$49.99" />
        <button @click="validateCode" class="btn-primary text-sm">Check</button>
      </div>
      <div v-if="validateResult" class="mt-3 p-3 rounded-xl text-sm"
        :class="validateResult.valid ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-red-50 text-red-700 border border-red-200'">
        <div v-if="validateResult.valid">
          <p class="font-semibold">✓ Valid — {{ validateResult.promotion?.description }}</p>
          <p class="mt-1">
            Discount:
            <strong>{{ validateResult.promotion?.discount_type === 'percent'
              ? validateResult.promotion.discount_value + '%'
              : '$' + validateResult.promotion?.discount_value }}
            </strong>
            <template v-if="testPrice > 0">
              → Saves <strong>${{ validateResult.discount_amount?.toFixed(2) }}</strong>
              → Final: <strong class="text-green-800">${{ validateResult.final_price?.toFixed(2) }}</strong>
            </template>
          </p>
        </div>
        <p v-else>{{ validateResult.error }}</p>
      </div>
    </div>

    <!-- Promos table -->
    <div class="card p-0 overflow-hidden">
      <table class="table">
        <thead>
          <tr>
            <th>Code</th>
            <th>Type</th>
            <th>Discount</th>
            <th>Plan</th>
            <th>Uses</th>
            <th>Valid Until</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="8" class="text-center py-8 text-gray-400">Loading…</td></tr>
          <tr v-else-if="!promos.length"><td colspan="8" class="text-center py-8 text-gray-400">No promotions created yet</td></tr>
          <tr v-for="p in promos" :key="p.id" :class="!p.is_active ? 'opacity-60' : ''">
            <td>
              <span class="font-mono font-bold text-blue-700 bg-blue-50 px-2 py-0.5 rounded-lg text-sm">{{ p.code }}</span>
            </td>
            <td class="text-xs capitalize text-gray-600">{{ p.discount_type }}</td>
            <td class="font-bold text-green-600">
              {{ p.discount_type === 'percent' ? p.discount_value + '%' : '$' + p.discount_value }}
            </td>
            <td class="text-xs text-gray-500">{{ p.plan_name || 'Any plan' }}</td>
            <td>
              <span class="text-sm">{{ p.uses_count }}</span>
              <span v-if="p.max_uses > 0" class="text-gray-400 text-xs"> / {{ p.max_uses }}</span>
              <span v-else class="text-gray-400 text-xs"> / ∞</span>
            </td>
            <td class="text-xs text-gray-400">
              {{ p.valid_until ? p.valid_until.slice(0,10) : 'No expiry' }}
            </td>
            <td>
              <span class="badge" :class="p.is_active ? 'badge-green' : 'badge-gray'">
                {{ p.is_active ? 'Active' : 'Off' }}
              </span>
            </td>
            <td>
              <div class="flex gap-1">
                <button @click="openEdit(p)" class="p-1.5 rounded hover:bg-gray-100 text-gray-400 hover:text-blue-600">
                  <PencilIcon class="w-4 h-4" />
                </button>
                <button v-if="authStore.isAdmin" @click="deletePromo(p)" class="p-1.5 rounded hover:bg-red-50 text-gray-400 hover:text-red-500">
                  <TrashIcon class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h3 class="text-lg font-semibold">{{ editing ? 'Edit' : 'New' }} Promotion</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600"><XMarkIcon class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="save" class="p-6 space-y-4">
          <div>
            <label class="form-label">Promo Code <span class="text-red-500">*</span></label>
            <input v-model="form.code" type="text" class="form-input font-mono uppercase" required
              placeholder="SUMMER20" :disabled="!!editing" />
            <p class="text-xs text-gray-400 mt-1">Code is automatically uppercase. Cannot be changed after creation.</p>
          </div>
          <div>
            <label class="form-label">Description</label>
            <input v-model="form.description" type="text" class="form-input" placeholder="Summer sale 20% off" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Discount Type</label>
              <select v-model="form.discount_type" class="form-input">
                <option value="percent">Percent (%)</option>
                <option value="fixed">Fixed Amount ($)</option>
              </select>
            </div>
            <div>
              <label class="form-label">Discount Value <span class="text-red-500">*</span></label>
              <input v-model.number="form.discount_value" type="number" step="0.01" min="0.01" class="form-input" required />
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Max Uses (0=unlimited)</label>
              <input v-model.number="form.max_uses" type="number" min="0" class="form-input" />
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Valid From</label>
              <input v-model="form.valid_from" type="datetime-local" class="form-input text-sm" />
            </div>
            <div>
              <label class="form-label">Valid Until</label>
              <input v-model="form.valid_until" type="datetime-local" class="form-input text-sm" />
            </div>
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="promo_active" v-model="form.is_active" class="rounded" />
            <label for="promo_active" class="text-sm text-gray-700">Promotion is active</label>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full spinner"></span>
              {{ editing ? 'Update' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { useAuthStore } from '@/store/auth'
import { promotionsAPI } from '@/api'
import { PlusIcon, XMarkIcon, TrashIcon, PencilIcon } from '@heroicons/vue/24/outline'
import axios from 'axios'

const toast = useToast()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const editing = ref(null)
const promos = ref([])
const testCode = ref('')
const testPrice = ref(0)
const validateResult = ref(null)
const form = reactive({ code: '', description: '', discount_type: 'percent', discount_value: 10, max_uses: 0, valid_from: '', valid_until: '', is_active: true })

async function load() {
  loading.value = true
  try { const { data } = await promotionsAPI.list(); promos.value = data.data || [] }
  catch { } finally { loading.value = false }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { code: '', description: '', discount_type: 'percent', discount_value: 10, max_uses: 0, valid_from: '', valid_until: '', is_active: true })
  showModal.value = true
}

function openEdit(p) {
  editing.value = p.id
  Object.assign(form, { code: p.code, description: p.description || '', discount_type: p.discount_type, discount_value: p.discount_value, max_uses: p.max_uses, valid_from: '', valid_until: '', is_active: p.is_active })
  showModal.value = true
}

async function save() {
  saving.value = true
  form.code = form.code.toUpperCase()
  try {
    const payload = { ...form, valid_from: form.valid_from || null, valid_until: form.valid_until || null }
    if (editing.value) { await promotionsAPI.update(editing.value, payload); toast.success('Updated') }
    else { await promotionsAPI.create(payload); toast.success('Created') }
    showModal.value = false; load()
  } catch (err) { toast.error(err.response?.data?.error || 'Failed') }
  finally { saving.value = false }
}

async function validateCode() {
  validateResult.value = null
  try {
    const { data } = await promotionsAPI.validate({ code: testCode.value, original_price: testPrice.value || 0 })
    validateResult.value = data
  } catch (err) {
    validateResult.value = { valid: false, error: err.response?.data?.error || 'Invalid code' }
  }
}

async function deletePromo(p) {
  if (!confirm(`Delete promo "${p.code}"?`)) return
  try { await promotionsAPI.delete(p.id); toast.success('Deleted'); load() }
  catch { toast.error('Failed') }
}

onMounted(load)
</script>
