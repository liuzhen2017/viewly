// API client: wraps fetch with the Viewly envelope ({code, msg, data}) and
// manages the guest/user JWT lifecycle in localStorage.
//
// Multi-tenant: the tenant slug comes from the host (first subdomain label in
// production) or a localStorage override for local testing; tokens are stored
// per tenant so switching sites never reuses a foreign-session token.

export function tenantSlug() {
  const saved = localStorage.getItem('viewly_tenant')
  if (saved) return saved
  const host = location.hostname
  if (host.endsWith('.localhost')) return host.slice(0, -'.localhost'.length) // demo.localhost → demo
  const parts = host.split('.')
  // e.g. tenant1.viewly.com → tenant1; viewly.com / localhost → main
  return parts.length > 2 ? parts[0] : 'main'
}

const TOKEN_KEY = 'viewly_token_' + tenantSlug()
const DEVICE_KEY = 'viewly_device'

export function getToken() { return localStorage.getItem(TOKEN_KEY) || '' }
export function setToken(t) { localStorage.setItem(TOKEN_KEY, t) }

function deviceID() {
  let id = localStorage.getItem(DEVICE_KEY)
  if (!id) {
    id = 'dev-' + Math.random().toString(36).slice(2, 10) + Date.now().toString(36)
    localStorage.setItem(DEVICE_KEY, id)
  }
  return id
}

// ensureGuest: on first boot, create an anonymous account so coins/tasks work
// before the visitor ever signs in.
export async function ensureAuth() {
  if (getToken()) return
  const r = await request('/api/auth/guest', { method: 'POST', body: { device_id: deviceID() } })
  setToken(r.token)
}

export async function request(path, { method = 'GET', body, auth = true } = {}) {
  const headers = { 'Content-Type': 'application/json', 'X-Tenant-Slug': tenantSlug() }
  if (auth && getToken()) headers.Authorization = 'Bearer ' + getToken()
  const res = await fetch(path, {
    method, headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  let json
  try { json = await res.json() } catch { throw new Error('bad response') }
  if (json.code !== 0) {
    const err = new Error(json.msg || 'error')
    err.code = json.code
    err.data = json.data
    throw err
  }
  return json.data
}

export const api = {
  // catalog
  home: () => request('/api/home'),
  categories: () => request('/api/categories'),
  dramas: (params = {}) => {
    const q = new URLSearchParams(params).toString()
    return request('/api/dramas' + (q ? '?' + q : ''))
  },
  drama: (id) => request(`/api/dramas/${id}`),
  search: (kw) => request('/api/search?keyword=' + encodeURIComponent(kw)),
  feed: (page = 1) => request(`/api/feed?page=${page}`),
  play: (id) => request(`/api/episodes/${id}/play`),
  store: () => request('/api/store'),

  // user
  me: () => request('/api/user/me'),
  guestLogin: () => request('/api/auth/guest', { method: 'POST', body: { device_id: deviceID() } }),
  register: (email, password) => request('/api/auth/register', { method: 'POST', body: { email, password } }),
  login: (email, password) => request('/api/auth/login', { method: 'POST', body: { email, password } }),
  bind: (email, password) => request('/api/auth/bind', { method: 'POST', body: { email, password } }),

  // engagement
  unlock: (id) => request(`/api/episodes/${id}/unlock`, { method: 'POST' }),
  progress: (id, position_sec, duration_sec) =>
    request(`/api/episodes/${id}/progress`, { method: 'POST', body: { position_sec, duration_sec } }),
  history: () => request('/api/history'),
  favorite: (id) => request(`/api/favorites/${id}`, { method: 'POST' }),
  like: (id) => request(`/api/likes/${id}`, { method: 'POST' }),
  follow: (id) => request(`/api/follows/${id}`, { method: 'POST' }),
  favorites: () => request('/api/favorites'),
  shareProgress: () => request('/api/rewards/tasks/share/progress', { method: 'POST' }),

  // rewards & wallet
  rewards: () => request('/api/rewards'),
  checkin: () => request('/api/rewards/checkin', { method: 'POST' }),
  claimTask: (key) => request(`/api/rewards/tasks/${key}/claim`, { method: 'POST' }),
  watchAdComplete: () => request('/api/rewards/watch-ad/complete', { method: 'POST' }),
  adConfig: () => request('/api/ad-config'),
  wallet: () => request('/api/wallet'),
  transactions: (page = 1) => request(`/api/wallet/transactions?page=${page}`),

  // orders
  createOrder: (kind, package_id) => request('/api/orders', { method: 'POST', body: { kind, package_id } }),
  mockPay: (orderNo) => request(`/api/orders/${orderNo}/mock-pay`, { method: 'POST' }),
}
