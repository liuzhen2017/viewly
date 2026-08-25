import { reactive } from 'vue'
import { api, ensureAuth, getToken, setToken } from './api'

// Global session store: current user, coin balance, helper to refresh.
export const store = reactive({
  user: null,
  ready: false,
  adConfig: null,
})

// injectAdSense loads the tenant's ad settings once per boot and, when the
// tenant enabled AdSense, mounts Google's script. Ad units render nothing
// until the site passes AdSense review — that's expected.
async function injectAdSense() {
  try {
    const cfg = await api.adConfig()
    store.adConfig = cfg
    if (cfg.adsense_enabled && !document.querySelector('script[data-adsense]')) {
      const s = document.createElement('script')
      s.async = true
      s.src = 'https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=' + cfg.adsense_client
      s.crossOrigin = 'anonymous'
      s.setAttribute('data-adsense', '1')
      document.head.appendChild(s)
    }
  } catch { /* ad config is optional */ }
}

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
  injectAdSense()
}

export function fmtCoins(n) {
  if (n === null || n === undefined) return '0'
  return Number(n).toLocaleString('en-US')
}
