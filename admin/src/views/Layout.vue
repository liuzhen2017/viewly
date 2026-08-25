<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { clearToken, getToken, getRole, tenantSlug } from '../api'
import { locale, setLocale, t } from '../i18n'

const router = useRouter()
const route = useRoute()
const isSuper = getRole() === 'super'
const menus = computed(() => {
  const base = [
    { path: '/dashboard', key: 'dashboard' },
    { path: '/dramas', key: 'dramas' },
    { path: '/categories', key: 'categories' },
    { path: '/banners', key: 'banners' },
    { path: '/packages', key: 'packages' },
    { path: '/users', key: 'users' },
    { path: '/orders', key: 'orders' },
  ]
  base.push({ path: '/ad-settings', key: 'adSettings' })
  if (isSuper) base.push({ path: '/tenants', key: 'tenantSite' })
  return base
})
function logout() {
  clearToken()
  router.replace('/login')
}
</script>

<template>
  <el-container style="height:100vh">
    <el-aside width="200px" style="background:#17171f">
      <div class="logo">🎬 {{ t('appName') }}</div>
      <div class="tenant-chip">{{ tenantSlug() }}</div>
      <div
        v-for="m in menus" :key="m.path" class="menu-item"
        :class="{ active: route.path === m.path }"
        @click="router.push(m.path)"
      >{{ t(m.key) }}</div>
    </el-aside>
    <el-container>
      <el-header style="display:flex;align-items:center;justify-content:flex-end;gap:12px;border-bottom:1px solid #eee">
        <div class="lang-switch">
          <button :class="{ on: locale === 'zh' }" @click="setLocale('zh')">中文</button>
          <button :class="{ on: locale === 'en' }" @click="setLocale('en')">EN</button>
        </div>
        <el-button size="small" @click="logout">{{ t('logout') }}</el-button>
      </el-header>
      <el-main style="background:#f5f6f8">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.logo { color: #fff; font-weight: 800; padding: 20px 16px 4px; font-size: 15px; }
.tenant-chip {
  color: #ffba99;
  font-size: 11px;
  padding: 0 16px 10px;
  border-bottom: 1px solid #262636;
  margin-bottom: 6px;
}
.tenant-chip::before { content: '🏬 '; }
.menu-item {
  color: #bbb; padding: 12px 20px; cursor: pointer; font-size: 14px;
}
.menu-item:hover { color: #fff; }
.menu-item.active { color: #fff; background: #ff5a3c; }
.lang-switch {
  display: inline-flex;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  overflow: hidden;
}
.lang-switch button {
  padding: 5px 12px;
  font-size: 12px;
  background: #fff;
  color: #606266;
  cursor: pointer;
}
.lang-switch button.on {
  background: #ff5a3c;
  color: #fff;
  font-weight: 700;
}
</style>
