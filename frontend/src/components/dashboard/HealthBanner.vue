<template>
  <div
    class="rounded-xl border px-4 py-3 flex items-start sm:items-center gap-3"
    :class="bannerClass"
  >
    <component :is="icon" class="w-5 h-5 flex-shrink-0 mt-0.5 sm:mt-0" />
    <div class="flex-1 min-w-0">
      <p class="text-sm font-semibold">{{ title }}</p>
      <p class="text-xs opacity-80 mt-0.5">{{ message }}</p>
    </div>
    <router-link
      v-if="health?.status !== 'healthy'"
      to="/security"
      class="text-xs font-medium underline flex-shrink-0"
    >
      View →
    </router-link>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { CheckCircleIcon, ExclamationTriangleIcon, XCircleIcon } from '@heroicons/vue/24/solid'

const props = defineProps({
  health: { type: Object, default: () => ({ status: 'healthy', messages: [] }) },
})

const bannerClass = computed(() => ({
  healthy: 'bg-green-50 border-green-200 text-green-800',
  degraded: 'bg-amber-50 border-amber-200 text-amber-800',
  critical: 'bg-red-50 border-red-200 text-red-800',
}[props.health?.status] || 'bg-green-50 border-green-200 text-green-800'))

const icon = computed(() => ({
  healthy: CheckCircleIcon,
  degraded: ExclamationTriangleIcon,
  critical: XCircleIcon,
}[props.health?.status] || CheckCircleIcon))

const title = computed(() => ({
  healthy: 'System Healthy',
  degraded: 'System Degraded',
  critical: 'Attention Required',
}[props.health?.status] || 'System Healthy'))

const message = computed(() => (props.health?.messages || []).join(' · ') || 'All systems operational')
</script>
