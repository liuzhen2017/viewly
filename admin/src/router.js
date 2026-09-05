import { createRouter, createWebHashHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/login', component: () => import('./views/Login.vue') },
    { path: '/', component: () => import('./views/Layout.vue'), children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', component: () => import('./views/Dashboard.vue') },
      { path: 'dramas', component: () => import('./views/Dramas.vue') },
      { path: 'categories', component: () => import('./views/Categories.vue') },
      { path: 'banners', component: () => import('./views/Banners.vue') },
      { path: 'packages', component: () => import('./views/Packages.vue') },
      { path: 'users', component: () => import('./views/Users.vue') },
      { path: 'orders', component: () => import('./views/Orders.vue') },
      { path: 'tenants', component: () => import('./views/Tenants.vue') },
      { path: 'tiktok', component: () => import('./views/TikTok.vue') },
      { path: 'ad-settings', component: () => import('./views/AdSettings.vue') },
    ] },
  ],
})
