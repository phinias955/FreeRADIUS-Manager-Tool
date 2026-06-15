<template>
  <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-2">
    <router-link
      v-for="action in actions"
      :key="action.to"
      :to="action.to"
      class="flex items-center gap-2.5 px-3 py-2.5 rounded-lg border border-gray-200 bg-white
             hover:border-blue-300 hover:bg-blue-50 transition-colors group"
    >
      <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" :class="action.bg">
        <component :is="action.icon" class="w-4 h-4" :class="action.color" />
      </div>
      <span class="text-xs font-medium text-gray-700 group-hover:text-blue-700">{{ action.label }}</span>
    </router-link>
  </div>
</template>

<script setup>
import {
  BeakerIcon, UserPlusIcon, SignalIcon, ShieldCheckIcon,
  ServerIcon, TicketIcon, MagnifyingGlassCircleIcon,
} from '@heroicons/vue/24/outline'

const actions = [
  { to: '/security/simulator', label: 'RADIUS Test', icon: BeakerIcon, bg: 'bg-indigo-100', color: 'text-indigo-600' },
  { to: '/users', label: 'Add User', icon: UserPlusIcon, bg: 'bg-purple-100', color: 'text-purple-600' },
  { to: '/monitor', label: 'Monitor', icon: SignalIcon, bg: 'bg-blue-100', color: 'text-blue-600' },
  { to: '/security', label: 'Security', icon: ShieldCheckIcon, bg: 'bg-red-100', color: 'text-red-600' },
  { to: '/nas', label: 'NAS Devices', icon: ServerIcon, bg: 'bg-orange-100', color: 'text-orange-600' },
  { to: '/network-scanner', label: 'Scan Network', icon: MagnifyingGlassCircleIcon, bg: 'bg-cyan-100', color: 'text-cyan-600' },
  { to: '/vouchers', label: 'Vouchers', icon: TicketIcon, bg: 'bg-teal-100', color: 'text-teal-600' },
]
</script>
