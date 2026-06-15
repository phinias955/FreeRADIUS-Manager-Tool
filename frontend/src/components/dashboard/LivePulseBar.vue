<template>
  <div class="rounded-xl bg-gradient-to-r from-slate-900 via-slate-800 to-slate-900 text-white px-4 py-3 overflow-x-auto">
    <div class="flex items-center gap-6 min-w-max">
      <div class="flex items-center gap-2 pr-4 border-r border-white/10">
        <span class="relative flex h-2 w-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full h-2 w-2 bg-green-400"></span>
        </span>
        <span class="text-xs font-semibold uppercase tracking-wider text-green-300">Live</span>
      </div>

      <div v-for="m in metrics" :key="m.label" class="text-center">
        <p class="text-[10px] uppercase tracking-wide text-slate-400">{{ m.label }}</p>
        <p class="text-sm font-bold tabular-nums" :class="m.warn ? 'text-amber-300' : 'text-white'">{{ m.value }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  live: Object,
  summary: Object,
})

const metrics = computed(() => {
  const live = props.live || {}
  const s = props.summary || {}
  return [
    { label: 'Auths (5m)', value: live.auth_last_5m ?? 0 },
    { label: 'Rejects (5m)', value: live.reject_last_5m ?? 0, warn: (live.reject_last_5m ?? 0) > 0 },
    { label: 'NAS Online', value: `${s.nas_up ?? 0}/${s.total_nas ?? 0}` },
    { label: '↓ In', value: `${(live.bandwidth_in_mbps ?? 0).toFixed(1)} Mbps` },
    { label: '↑ Out', value: `${(live.bandwidth_out_mbps ?? 0).toFixed(1)} Mbps` },
    { label: 'Blocked IPs', value: s.blocked_ips ?? 0, warn: (s.blocked_ips ?? 0) > 0 },
    { label: 'Honeypot Today', value: s.honeypot_today ?? 0 },
    ...(s.peak_hour ? [{ label: 'Peak Hour', value: `${s.peak_hour} (${s.peak_hour_count})` }] : []),
  ]
})
</script>
