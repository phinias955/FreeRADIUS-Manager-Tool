<template>
  <div class="stat-card !p-4">
    <div class="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0" :class="iconBg">
      <component :is="iconComponent" class="w-5 h-5" :class="iconColor" />
    </div>
    <div class="min-w-0">
      <p class="text-[10px] font-medium text-gray-500 uppercase tracking-wide truncate">{{ title }}</p>
      <div v-if="loading" class="h-6 w-12 bg-gray-200 rounded animate-pulse mt-1"></div>
      <p v-else class="text-xl font-bold text-gray-900 truncate">{{ value }}</p>
      <p v-if="trend != null && !loading" class="text-[10px] mt-0.5 font-medium" :class="trend >= 0 ? 'text-green-600' : 'text-red-600'">
        {{ trend >= 0 ? '↑' : '↓' }} {{ Math.abs(trend).toFixed(1) }}% vs yesterday
      </p>
      <p v-else-if="subtitle && !loading" class="text-[10px] text-gray-400 mt-0.5 truncate">{{ subtitle }}</p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import {
  SignalIcon, UsersIcon, ServerIcon, ShieldCheckIcon,
  CheckCircleIcon, XCircleIcon, BoltIcon, ChartBarIcon,
} from '@heroicons/vue/24/outline'

const props = defineProps({
  title: String,
  value: [String, Number],
  subtitle: String,
  trend: Number,
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
  indigo: { bg: 'bg-indigo-100', icon: 'text-indigo-600' },
  teal: { bg: 'bg-teal-100', icon: 'text-teal-600' },
  cyan: { bg: 'bg-cyan-100', icon: 'text-cyan-600' },
}

const iconMap = {
  signal: SignalIcon,
  users: UsersIcon,
  'user-check': ShieldCheckIcon,
  server: ServerIcon,
  check: CheckCircleIcon,
  x: XCircleIcon,
  bolt: BoltIcon,
  chart: ChartBarIcon,
}

const iconBg = computed(() => colorMap[props.color]?.bg || 'bg-gray-100')
const iconColor = computed(() => colorMap[props.color]?.icon || 'text-gray-600')
const iconComponent = computed(() => iconMap[props.icon] || UsersIcon)
</script>
