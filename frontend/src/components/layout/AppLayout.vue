<template>
  <div class="flex h-screen bg-gray-50 overflow-hidden">
    <!-- Sidebar -->
    <aside
      class="flex-shrink-0 w-64 bg-white border-r border-gray-200 flex flex-col"
      :class="{ '-translate-x-full': !sidebarOpen }"
    >
      <!-- Logo -->
      <div class="flex items-center gap-3 px-5 py-4 border-b border-gray-200">
        <div class="w-9 h-9 bg-blue-600 rounded-lg flex items-center justify-center flex-shrink-0">
          <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
          </svg>
        </div>
        <div>
          <div class="font-bold text-gray-900 text-sm leading-tight">RADIUS Manager Pro</div>
          <div class="text-xs text-blue-600 font-medium">v2.0 Pro</div>
        </div>
      </div>

      <!-- Nav -->
      <nav class="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        <router-link to="/dashboard" class="sidebar-link" active-class="active">
          <HomeIcon class="w-5 h-5 flex-shrink-0" />
          Dashboard
        </router-link>

        <router-link to="/users" class="sidebar-link" active-class="active">
          <UsersIcon class="w-5 h-5 flex-shrink-0" />
          RADIUS Users
        </router-link>

        <router-link to="/nas" class="sidebar-link" active-class="active">
          <ServerIcon class="w-5 h-5 flex-shrink-0" />
          NAS Devices
        </router-link>

        <router-link to="/monitor" class="sidebar-link" active-class="active">
          <ChartBarIcon class="w-5 h-5 flex-shrink-0" />
          Monitoring
        </router-link>

        <router-link to="/vouchers" class="sidebar-link" active-class="active">
          <TicketIcon class="w-5 h-5 flex-shrink-0" />
          Vouchers
        </router-link>

        <router-link to="/reports" class="sidebar-link" active-class="active">
          <DocumentChartBarIcon class="w-5 h-5 flex-shrink-0" />
          Reports
        </router-link>

        <template v-if="authStore.isAdmin || authStore.isSuperAdmin">
          <div class="pt-4 pb-2">
            <p class="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Business</p>
          </div>
          <router-link to="/plans" class="sidebar-link" active-class="active">
            <CurrencyDollarIcon class="w-5 h-5 flex-shrink-0" />
            User Plans
          </router-link>
          <router-link to="/billing" class="sidebar-link" active-class="active">
            <DocumentTextIcon class="w-5 h-5 flex-shrink-0" />
            Billing
          </router-link>
          <router-link to="/alerts" class="sidebar-link" active-class="active">
            <BellAlertIcon class="w-5 h-5 flex-shrink-0" />
            Alert Rules
          </router-link>
        </template>

        <template v-if="authStore.isAdmin || authStore.isSuperAdmin">
          <div class="pt-4 pb-2">
            <p class="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Network</p>
          </div>
          <router-link to="/bandwidth" class="sidebar-link" active-class="active">
            <SignalIcon class="w-5 h-5 flex-shrink-0" />
            Bandwidth Profiles
          </router-link>
        </template>

        <template v-if="authStore.isSuperAdmin">
          <div class="pt-4 pb-2">
            <p class="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Admin</p>
          </div>
          
          <router-link to="/admin-users" class="sidebar-link" active-class="active">
            <ShieldCheckIcon class="w-5 h-5 flex-shrink-0" />
            Admin Users
          </router-link>

          <router-link to="/settings" class="sidebar-link" active-class="active">
            <CogIcon class="w-5 h-5 flex-shrink-0" />
            Settings
          </router-link>
        </template>
      </nav>

      <!-- User info -->
      <div class="border-t border-gray-200 p-4">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-full bg-blue-100 flex items-center justify-center flex-shrink-0">
            <span class="text-blue-700 font-semibold text-sm">
              {{ userInitials }}
            </span>
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-gray-900 truncate">{{ authStore.user?.full_name || authStore.user?.username }}</p>
            <p class="text-xs text-gray-500 capitalize">{{ authStore.user?.role?.replace('_', ' ') }}</p>
          </div>
          <button
            @click="handleLogout"
            class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
            title="Sign out"
          >
            <ArrowRightOnRectangleIcon class="w-4 h-4" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Main content -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Top bar -->
      <header class="bg-white border-b border-gray-200 px-6 py-3 flex items-center justify-between flex-shrink-0">
        <div class="flex items-center gap-4">
          <h1 class="font-semibold text-gray-900">{{ currentPageTitle }}</h1>
        </div>

        <div class="flex items-center gap-3">
          <!-- Health indicator -->
          <div class="flex items-center gap-1.5 text-xs text-gray-500">
            <span
              class="w-2 h-2 rounded-full"
              :class="systemHealthy ? 'bg-green-500' : 'bg-red-500'"
            ></span>
            {{ systemHealthy ? 'System OK' : 'System Error' }}
          </div>

          <div class="h-5 w-px bg-gray-200"></div>

          <span class="text-xs text-gray-500">{{ currentTime }}</span>
        </div>
      </header>

      <!-- Page content -->
      <main class="flex-1 overflow-y-auto">
        <div class="p-6">
          <router-view v-slot="{ Component }">
            <transition name="fade" mode="out-in">
              <component :is="Component" />
            </transition>
          </router-view>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { useToast } from 'vue-toastification'
import { healthAPI } from '@/api'
import {
  HomeIcon,
  UsersIcon,
  ServerIcon,
  ChartBarIcon,
  ShieldCheckIcon,
  CogIcon,
  ArrowRightOnRectangleIcon,
  TicketIcon,
  DocumentChartBarIcon,
  SignalIcon,
  CurrencyDollarIcon,
  DocumentTextIcon,
  BellAlertIcon,
} from '@heroicons/vue/24/outline'

const authStore = useAuthStore()
const router = useRouter()
const route = useRoute()
const toast = useToast()

const sidebarOpen = ref(true)
const systemHealthy = ref(true)
const currentTime = ref('')

const userInitials = computed(() => {
  const name = authStore.user?.full_name || authStore.user?.username || ''
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

const currentPageTitle = computed(() => {
  return route.meta?.title || 'FreeRADIUS Manager'
})

async function checkHealth() {
  try {
    await healthAPI.check()
    systemHealthy.value = true
  } catch {
    systemHealthy.value = false
  }
}

function updateTime() {
  currentTime.value = new Date().toLocaleTimeString()
}

async function handleLogout() {
  await authStore.logout()
  toast.success('Logged out successfully')
  router.push('/login')
}

let healthTimer, clockTimer

onMounted(() => {
  checkHealth()
  updateTime()
  healthTimer = setInterval(checkHealth, 30000)
  clockTimer = setInterval(updateTime, 1000)
})

onUnmounted(() => {
  clearInterval(healthTimer)
  clearInterval(clockTimer)
})
</script>
