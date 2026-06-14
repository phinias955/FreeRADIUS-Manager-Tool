import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/store/auth'

const routes = [
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

router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()

  if (to.meta.title) {
    document.title = `${to.meta.title} — FreeRADIUS Manager`
  }

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
