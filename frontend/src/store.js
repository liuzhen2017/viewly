import { reactive } from 'vue'
import { api, ensureAuth, getToken, setToken } from './api'

// Global session store: current user, coin balance, helper to refresh.
export const store = reactive({
  user: null,
  ready: false,
})

export async function refreshMe() {
  try {
    store.user = await api.me()
  } catch {
    store.user = null
  }
}

export async function boot() {
  if (!getToken()) {
    try {
      const r = await api.guestLogin()
      setToken(r.token)
    } catch { /* offline: pages degrade to anonymous */ }
  }
  await refreshMe()
  store.ready = true
}

export function fmtCoins(n) {
  if (n === null || n === undefined) return '0'
  return Number(n).toLocaleString('en-US')
}
