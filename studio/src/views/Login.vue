<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api, setToken, getToken } from '../api'
import { t, locale, setLocale } from '../i18n'

const router = useRouter()
const email = ref('')
const password = ref('')
const mode = ref('login') // login | register
const loading = ref(false)

onMounted(() => {
  if (getToken()) router.replace('/dashboard')
})

async function submit() {
  if (!email.value || password.value.length < 6) {
    ElMessage.warning(t('email') + ' / ' + t('password') + ' (6+)')
    return
  }
  loading.value = true
  try {
    const fn = mode.value === 'login' ? api.login : api.register
    const r = await fn(email.value.trim(), password.value)
    setToken(r.token)
    router.replace('/dashboard')
  } catch (e) {
    ElMessage.error((mode.value === 'login' ? t('loginFailed') : t('regFailed')) + ': ' + e.message)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background: #f5f6f8">
    <el-card style="width: 380px">
      <template #header>
        <div style="display: flex; align-items: center; justify-content: space-between">
          <span style="font-weight: 700; color: #ff5a3c">FoxDrama Studio</span>
          <el-select :model-value="locale" size="small" style="width: 80px" @change="(v) => setLocale(v)">
            <el-option value="en" label="EN" />
            <el-option value="zh" label="中文" />
          </el-select>
        </div>
      </template>
      <el-form @submit.prevent="submit">
        <el-form-item :label="t('email')">
          <el-input v-model="email" type="email" autocomplete="username" />
        </el-form-item>
        <el-form-item :label="t('password')">
          <el-input v-model="password" type="password" show-password autocomplete="current-password"
                    @keyup.enter="submit" />
        </el-form-item>
        <el-button type="primary" style="width: 100%" :loading="loading" @click="submit">
          {{ mode === 'login' ? t('signIn') : t('signUp') }}
        </el-button>
        <div style="margin-top: 12px; text-align: center">
          <el-link type="primary" @click="mode = mode === 'login' ? 'register' : 'login'">
            {{ mode === 'login' ? t('noAccount') : t('haveAccount') }}
          </el-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>
