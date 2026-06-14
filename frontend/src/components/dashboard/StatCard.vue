<template>
  <div class="stat-card">
    <div class="w-12 h-12 rounded-xl flex items-center justify-center flex-shrink-0" :class="iconBg">
      <component :is="iconComponent" class="w-6 h-6" :class="iconColor" />
    </div>
    <div>
      <p class="text-xs font-medium text-gray-500 uppercase tracking-wide">{{ title }}</p>
      <div v-if="loading" class="h-7 w-16 bg-gray-200 rounded animate-pulse mt-1"></div>
      <p v-else class="text-2xl font-bold text-gray-900">{{ value }}</p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { SignalIcon, UsersIcon, UserIcon, ServerIcon, ShieldCheckIcon } from '@heroicons/vue/24/outline'

const props = defineProps({
  title: String,
  value: [String, Number],
  icon: String,
  color: { type: String, default: 'blue' },
  loading: Boolean,
})

const colorMap = {
  blue: { bg: 'bg-blue-100', icon: 'text-blue-600' },
  purple: { bg: 'bg-purple-100', icon: 'text-purple-600' },
  green: { bg: 'bg-green-100', icon: 'text-green-600' },
  orange: { bg: 'bg-orange-100', icon: 'text-orange-600' },
  red: { bg: 'bg-red-100', icon: 'text-red-600' },
}

const iconMap = {
  signal: SignalIcon,
  users: UsersIcon,
  'user-check': ShieldCheckIcon,
  server: ServerIcon,
  user: UserIcon,
}

const iconBg = computed(() => colorMap[props.color]?.bg || 'bg-gray-100')
const iconColor = computed(() => colorMap[props.color]?.icon || 'text-gray-600')
const iconComponent = computed(() => iconMap[props.icon] || UserIcon)
</script>
