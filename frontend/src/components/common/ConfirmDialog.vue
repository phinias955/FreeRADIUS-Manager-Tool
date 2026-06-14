<template>
  <div class="modal-overlay" @click.self="$emit('cancel')">
    <transition name="slide-up" appear>
      <div class="modal max-w-sm">
        <div class="modal-header">
          <h2 class="text-lg font-semibold text-gray-900">{{ title }}</h2>
          <button @click="$emit('cancel')" class="p-1.5 hover:bg-gray-100 rounded-lg">
            <XMarkIcon class="w-5 h-5 text-gray-500" />
          </button>
        </div>
        <div class="modal-body">
          <p class="text-sm text-gray-600">{{ message }}</p>
        </div>
        <div class="modal-footer">
          <button @click="$emit('cancel')" class="btn-secondary">Cancel</button>
          <button @click="$emit('confirm')" :class="confirmClass || 'btn-danger'">
            {{ confirmLabel || 'Confirm' }}
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { XMarkIcon } from '@heroicons/vue/24/outline'
defineProps({ title: String, message: String, confirmLabel: String, confirmClass: String })
defineEmits(['confirm', 'cancel'])
</script>
