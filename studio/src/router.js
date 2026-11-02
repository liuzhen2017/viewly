import { createRouter, createWebHashHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/login', component: () => import('./views/Login.vue') },
    {
      path: '/',
      component: () => import('./views/Layout.vue'),
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', component: () => import('./views/Dashboard.vue') },
        { path: 'dramas', component: () => import('./views/Dramas.vue') },
        { path: 'clips', component: () => import('./views/Clips.vue') },
        { path: 'accounts', component: () => import('./views/Accounts.vue') },
        { path: 'new-task', component: () => import('./views/NewTask.vue') },
        { path: 'tasks', component: () => import('./views/Tasks.vue') },
      ],
    },
  ],
})
