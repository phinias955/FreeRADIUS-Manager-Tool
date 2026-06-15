<template>
  <div class="h-full flex flex-col items-center justify-center text-center px-4">
    <ChartBarIcon class="w-10 h-10 text-gray-300 mb-2" />
    <p class="text-sm text-gray-400 max-w-xs">{{ message }}</p>
  </div>
</template>

<script setup>
import { ChartBarIcon } from '@heroicons/vue/24/outline'

defineProps({
  message: { type: String, default: 'No data available' },
})
</script>
