<template>
  <div v-if="password" class="space-y-1.5">
    <div class="flex gap-1">
      <div
        v-for="i in 4"
        :key="i"
        class="h-1.5 flex-1 rounded-full transition-colors duration-200"
        :class="i <= score ? scoreColor : 'bg-gray-200'"
      ></div>
    </div>
    <p class="text-xs" :class="scoreTextColor">
      {{ scoreLabel }}
      <span v-if="hint" class="text-gray-500 ml-1">— {{ hint }}</span>
    </p>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({ password: String })

const score = computed(() => {
  const pw = props.password || ''
  let s = 0
  if (pw.length >= 12) s++
  if (/[A-Z]/.test(pw)) s++
  if (/[0-9]/.test(pw)) s++
  if (/[^A-Za-z0-9]/.test(pw)) s++
  return s
})

const scoreColor = computed(() => ['bg-red-500', 'bg-orange-500', 'bg-yellow-500', 'bg-green-500'][score.value - 1] || 'bg-red-500')
const scoreTextColor = computed(() => ['text-red-600', 'text-orange-600', 'text-yellow-600', 'text-green-600'][score.value - 1] || 'text-red-600')
const scoreLabel = computed(() => ['Very Weak', 'Weak', 'Fair', 'Strong'][score.value - 1] || 'Too Short')

const hint = computed(() => {
  const pw = props.password || ''
  if (pw.length < 12) return 'needs 12+ characters'
  if (!/[A-Z]/.test(pw)) return 'add uppercase letter'
  if (!/[0-9]/.test(pw)) return 'add a number'
  if (!/[^A-Za-z0-9]/.test(pw)) return 'add special character'
  return ''
})
</script>
