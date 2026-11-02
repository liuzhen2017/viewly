<script setup>
import { onMounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api, getToken, clearToken } from '../api'
import { t, locale, setLocale } from '../i18n'

const router = useRouter()
const route = useRoute()
const me = ref(null)

const nav = [
  { path: '/dashboard', key: 'navDashboard' },
  { path: '/dramas', key: 'navDramas' },
  { path: '/clips', key: 'navClips' },
  { path: '/accounts', key: 'navAccounts' },
  { path: '/new-task', key: 'navNewTask' },
  { path: '/tasks', key: 'navTasks' },
]

onMounted(async () => {
  if (!getToken()) {
    router.replace('/login')
    return
  }
  try {
    me.value = await api.me()
  } catch {
    router.replace('/login')
  }
})

function logout() {
  clearToken()
  ElMessage.success(t('logout'))
  router.replace('/login')
}
</script>

<template>
  <el-container style="min-height: 100vh">
    <el-aside width="200px" style="background: #17171f; display: flex; flex-direction: column">
      <div style="color: #ff5a3c; font-weight: 700; font-size: 18px; padding: 20px 16px">
        FoxDrama Studio
      </div>
      <el-menu :default-active="route.path" router background-color="#17171f" text-color="#bbb"
               active-text-color="#ff5a3c" style="border-right: none; flex: 1">
        <el-menu-item v-for="item in nav" :key="item.path" :index="item.path">
          {{ t(item.key) }}
        </el-menu-item>
      </el-menu>
      <div style="padding: 12px 16px; color: #888; font-size: 12px">
        {{ me ? (me.email || me.nickname || ('#' + me.id)) : '' }}
      </div>
    </el-aside>
    <el-container>
      <el-header style="display: flex; align-items: center; justify-content: flex-end; gap: 12px; border-bottom: 1px solid #eee">
        <el-select :model-value="locale" size="small" style="width: 90px"
                   @change="(v) => setLocale(v)">
          <el-option value="en" label="EN" />
          <el-option value="zh" label="中文" />
        </el-select>
        <el-button size="small" @click="logout">{{ t('logout') }}</el-button>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>
