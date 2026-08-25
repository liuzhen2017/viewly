<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, setToken } from '../api'
import { store, refreshMe } from '../store'

const router = useRouter()
const mode = ref(store.user?.email ? 'login' : 'register')
const email = ref('')
const password = ref('')
const busy = ref(false)
const err = ref('')

async function submit() {
  err.value = ''
  if (!email.value || password.value.length < 6) {
    err.value = 'Email + password (6+ chars) required'
    return
  }
  busy.value = true
  try {
    let r
    if (store.user && !store.user.email && mode.value === 'register') {
      // guest upgrading: bind email so coins carry over
      r = await api.bind(email.value, password.value)
      window.$toast('Email bound — your coins are safe now')
    } else {
      r = await (mode.value === 'register' ? api.register(email.value, password.value) : api.login(email.value, password.value))
      if (r.token) setToken(r.token)
    }
    await refreshMe()
    router.replace('/profile')
  } catch (e) {
    err.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="page">
    <button class="back" @click="$router.back()">‹</button>
    <div class="hero">
      <div class="logo">🎬</div>
      <h2>Welcome to Viewly</h2>
      <p>{{ store.user && !store.user.email ? 'Bind an email to protect your account' : 'Sign in to protect your account' }}</p>
    </div>

    <div class="form card">
      <div class="mode" v-if="!(store.user && !store.user.email)">
        <button :class="{ on: mode === 'login' }" @click="mode = 'login'">Sign In</button>
        <button :class="{ on: mode === 'register' }" @click="mode = 'register'">Register</button>
      </div>
      <input v-model="email" type="email" placeholder="Email" autocomplete="email" />
      <input v-model="password" type="password" placeholder="Password (6+ chars)" autocomplete="current-password" @keyup.enter="submit" />
      <div v-if="err" class="err">{{ err }}</div>
      <button class="btn btn-primary" style="width:100%;padding:13px;font-size:15px" :disabled="busy" @click="submit">
        {{ store.user && !store.user.email ? 'Bind Email' : mode === 'register' ? 'Create Account' : 'Sign In With Email' }}
      </button>
      <p class="tip">By continuing you agree to our User Agreement & Privacy Policy.</p>
    </div>
  </div>
</template>

<style scoped>
.back { font-size: 26px; color: var(--muted); }
.hero { text-align: center; padding: 30px 0 24px; }
.logo { font-size: 52px; }
.hero h2 { margin: 12px 0 6px; font-size: 21px; }
.hero p { color: var(--muted); font-size: 13px; margin: 0; }
.form { display: flex; flex-direction: column; gap: 12px; }
.mode { display: flex; background: var(--bg-elev2); border-radius: 10px; padding: 4px; }
.mode button { flex: 1; padding: 9px; border-radius: 8px; font-weight: 700; color: var(--muted); }
.mode button.on { background: var(--bg); color: var(--text); }
.err { color: #ff6b6b; font-size: 12.5px; }
.tip { color: var(--muted); font-size: 11px; text-align: center; margin: 4px 0 0; }
</style>
