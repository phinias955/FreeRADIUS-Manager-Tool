import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { setupAPI } from '@/api/index'

// Cache setup status so we only check once per page load
let setupRequired = null

async function checkSetupStatus() {
  if (setupRequired !== null) return setupRequired
  try {
    const { data } = await setupAPI.status()
    setupRequired = data.setup_required === true
  } catch {
    setupRequired = false
  }
  return setupRequired
}

// Called by SetupWizard after successful completion — prevents any back-navigation
export function markSetupComplete() {
  setupRequired = false
}

const routes = [
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/views/SetupWizard.vue'),
    meta: { requiresAuth: false, title: 'Setup Wizard' },
  },
  {
    path: '/portal',
    name: 'UserPortal',
    component: () => import('@/views/UserPortal.vue'),
    meta: { requiresAuth: false, title: 'My Account' },
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false, title: 'Sign In' },
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: '/dashboard',
      },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: 'Dashboard', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/Profile.vue'),
        meta: { title: 'My Profile', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/Users.vue'),
        meta: { title: 'RADIUS Users', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'nas',
        name: 'NAS',
        component: () => import('@/views/NASDevices.vue'),
        meta: { title: 'NAS Devices', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'monitor',
        name: 'Monitor',
        component: () => import('@/views/Monitor.vue'),
        meta: { title: 'Monitoring', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'vouchers',
        name: 'Vouchers',
        component: () => import('@/views/Vouchers.vue'),
        meta: { title: 'Vouchers', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'bandwidth',
        name: 'Bandwidth',
        component: () => import('@/views/BandwidthProfiles.vue'),
        meta: { title: 'Bandwidth Profiles', roles: ['admin', 'super_admin'] },
      },
      {
        path: 'reports',
        name: 'Reports',
        component: () => import('@/views/Reports.vue'),
        meta: { title: 'Reports', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'plans',
        name: 'Plans',
        component: () => import('@/views/Plans.vue'),
        meta: { title: 'User Plans', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'billing',
        name: 'Billing',
        component: () => import('@/views/Billing.vue'),
        meta: { title: 'Billing', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'alerts',
        name: 'AlertRules',
        component: () => import('@/views/AlertRules.vue'),
        meta: { title: 'Alert Rules', roles: ['admin', 'super_admin'] },
      },
      {
        path: 'ip-pools',
        name: 'IPPools',
        component: () => import('@/views/IPPools.vue'),
        meta: { title: 'IP Pools', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'api-keys',
        name: 'APIKeys',
        component: () => import('@/views/APIKeys.vue'),
        meta: { title: 'API Keys', roles: ['super_admin'] },
      },
      {
        path: 'scheduler',
        name: 'Scheduler',
        component: () => import('@/views/Scheduler.vue'),
        meta: { title: 'Scheduler', roles: ['admin', 'super_admin'] },
      },
      // ── Tier 4 Pro ──────────────────────────────────────────────────────
      {
        path: 'zones',
        name: 'HotspotZones',
        component: () => import('@/views/HotspotZones.vue'),
        meta: { title: 'Hotspot Zones', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'network-map',
        name: 'NetworkMap',
        component: () => import('@/views/NetworkMap.vue'),
        meta: { title: 'Network Map', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'sms',
        name: 'SMS',
        component: () => import('@/views/SMS.vue'),
        meta: { title: 'SMS Notifications', roles: ['admin', 'super_admin'] },
      },
      // ── Tier 5 Pro ──────────────────────────────────────────────────────
      {
        path: 'organizations',
        name: 'Organizations',
        component: () => import('@/views/Organizations.vue'),
        meta: { title: 'Organizations', roles: ['admin', 'super_admin'] },
      },
      {
        path: 'customers',
        name: 'Customers',
        component: () => import('@/views/Customers.vue'),
        meta: { title: 'Customers', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'tickets',
        name: 'Tickets',
        component: () => import('@/views/Tickets.vue'),
        meta: { title: 'Support Tickets', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'captive-portal',
        name: 'CaptivePortal',
        component: () => import('@/views/CaptivePortal.vue'),
        meta: { title: 'Captive Portals', roles: ['admin', 'super_admin'] },
      },
      {
        path: 'webhooks',
        name: 'Webhooks',
        component: () => import('@/views/Webhooks.vue'),
        meta: { title: 'Webhooks', roles: ['admin', 'super_admin'] },
      },
      // ── Tier 6 Pro ──────────────────────────────────────────────────────
      {
        path: 'payments',
        name: 'Payments',
        component: () => import('@/views/Payments.vue'),
        meta: { title: 'Payments', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'templates',
        name: 'RadiusTemplates',
        component: () => import('@/views/RadiusTemplates.vue'),
        meta: { title: 'RADIUS Templates', roles: ['admin', 'super_admin'] },
      },
      {
        path: 'promotions',
        name: 'Promotions',
        component: () => import('@/views/Promotions.vue'),
        meta: { title: 'Promotions', roles: ['admin', 'super_admin'] },
      },
      {
        path: 'bulk',
        name: 'BulkOps',
        component: () => import('@/views/BulkOps.vue'),
        meta: { title: 'Bulk Operations', roles: ['admin', 'super_admin'] },
      },
      // ── Tier 7: Security Suite ──────────────────────────────────────────
      {
        path: 'security',
        name: 'SecurityCenter',
        component: () => import('@/views/SecurityCenter.vue'),
        meta: { title: 'Security Center', roles: ['operator', 'admin', 'super_admin'] },
      },
      {
        path: 'security/honeypot',
        name: 'HoneypotLogs',
        component: () => import('@/views/HoneypotLogs.vue'),
        meta: { title: 'Honeypot Logs', roles: ['admin', 'super_admin'] },
      },
      {
        path: 'security/simulator',
        name: 'RadiusSimulator',
        component: () => import('@/views/RadiusSimulator.vue'),
        meta: { title: 'RADIUS Simulator', roles: ['admin', 'super_admin'] },
      },
      {
        path: 'admin-users',
        name: 'AdminUsers',
        component: () => import('@/views/AdminUsers.vue'),
        meta: { title: 'Admin Users', roles: ['super_admin'] },
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/Settings.vue'),
        meta: { title: 'Settings', roles: ['super_admin'] },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/dashboard',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  if (to.meta.title) {
    document.title = `${to.meta.title} — FreeRADIUS Manager`
  }

  // Always check setup status first
  const needsSetup = await checkSetupStatus()

  if (needsSetup) {
    // Setup not done — only /setup is allowed
    if (to.name !== 'Setup') return next({ name: 'Setup' })
    return next()
  }

  // Setup done — /setup is no longer accessible
  if (to.name === 'Setup') return next({ name: 'Login' })

  if (to.meta.requiresAuth === false) {
    if (authStore.isAuthenticated && to.name === 'Login') {
      return next('/dashboard')
    }
    return next()
  }

  if (!authStore.isAuthenticated) {
    return next({ name: 'Login', query: { redirect: to.fullPath } })
  }

  if (to.meta.roles && !to.meta.roles.includes(authStore.userRole)) {
    return next('/dashboard')
  }

  next()
})

export default router
