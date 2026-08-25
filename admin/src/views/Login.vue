<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { admin, setToken, setRole } from '../api'
import { locale, setLocale, t } from '../i18n'

const router = useRouter()
const username = ref('admin')
const password = ref('')
const busy = ref(false)

async function submit() {
  if (!username.value || !password.value) return
  busy.value = true
  try {
    const r = await admin.login(username.value, password.value)
    setToken(r.token)
    setRole(r.admin?.role || 'admin')
    router.replace('/dashboard')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <el-card class="login-card">
      <h2>🎬 {{ t('appName') }}</h2>
      <el-input v-model="username" :placeholder="t('username')" size="large" />
      <el-input v-model="password" type="password" :placeholder="t('password')" size="large" show-password @keyup.enter="submit" />
      <el-button type="primary" size="large" style="width:100%" :loading="busy" @click="submit">{{ t('signIn') }}</el-button>
      <div style="display:flex;justify-content:center">
        <div class="lang-switch">
          <button :class="{ on: locale === 'zh' }" @click="setLocale('zh')">中文</button>
          <button :class="{ on: locale === 'en' }" @click="setLocale('en')">EN</button>
        </div>
      </div>
      <p class="hint">{{ t('loginHint') }}</p>
    </el-card>
  </div>
</template>

<style scoped>
.login-wrap {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #17171f;
}
.login-card { width: 360px; display: flex; flex-direction: column; gap: 14px; }
.login-card h2 { margin: 0 0 6px; text-align: center; }
.hint { color: #999; font-size: 12px; text-align: center; margin: 0; }
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
