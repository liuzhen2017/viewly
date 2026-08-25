// Multi-tenant admin: X-Tenant-Slug selects the tenant site being managed
// (subdomain in production, localStorage override for local testing).
// Tokens and the admin role are stored per tenant.
export function tenantSlug() {
  const saved = localStorage.getItem('viewly_tenant')
  if (saved) return saved
  const host = location.hostname
  if (host.endsWith('.localhost')) return host.slice(0, -'.localhost'.length)
  const parts = host.split('.')
  return parts.length > 2 ? parts[0] : 'main'
}

const TOKEN_KEY = 'viewly_admin_token_' + tenantSlug()
const ROLE_KEY = 'viewly_admin_role_' + tenantSlug()
export const getToken = () => localStorage.getItem(TOKEN_KEY) || ''
export const setToken = (t) => localStorage.setItem(TOKEN_KEY, t)
export const getRole = () => localStorage.getItem(ROLE_KEY) || ''
export const setRole = (r) => localStorage.setItem(ROLE_KEY, r)
export const clearToken = () => { localStorage.removeItem(TOKEN_KEY); localStorage.removeItem(ROLE_KEY) }

export async function req(path, { method = 'GET', body } = {}) {
  const headers = { 'Content-Type': 'application/json', 'X-Tenant-Slug': tenantSlug() }
  if (getToken()) headers.Authorization = 'Bearer ' + getToken()
  const res = await fetch(path, {
    method, headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401) {
    clearToken()
    location.hash = '#/login'
    throw new Error('unauthorized')
  }
  const json = await res.json()
  if (json.code !== 0) throw new Error(json.msg || 'error')
  return json.data
}

export const admin = {
  login: (username, password) => req('/api/admin/login', { method: 'POST', body: { username, password } }),
  stats: () => req('/api/admin/stats'),

  dramas: (page = 1, keyword = '') => req(`/api/admin/dramas?page=${page}&size=20&keyword=${encodeURIComponent(keyword)}`),
  createDrama: (d) => req('/api/admin/dramas', { method: 'POST', body: d }),
  updateDrama: (id, d) => req(`/api/admin/dramas/${id}`, { method: 'PUT', body: d }),
  deleteDrama: (id) => req(`/api/admin/dramas/${id}`, { method: 'DELETE' }),

  episodes: (dramaId) => req(`/api/admin/episodes?drama_id=${dramaId}`),
  saveEpisode: (e) => e.id ? req(`/api/admin/episodes/${e.id}`, { method: 'PUT', body: e }) : req('/api/admin/episodes', { method: 'POST', body: e }),
  deleteEpisode: (id) => req(`/api/admin/episodes/${id}`, { method: 'DELETE' }),

  categories: () => req('/api/admin/categories'),
  saveCategory: (c) => req('/api/admin/categories', { method: 'POST', body: c }),
  deleteCategory: (id) => req(`/api/admin/categories/${id}`, { method: 'DELETE' }),

  banners: () => req('/api/admin/banners'),
  saveBanner: (b) => req('/api/admin/banners', { method: 'POST', body: b }),
  deleteBanner: (id) => req(`/api/admin/banners/${id}`, { method: 'DELETE' }),

  packages: () => req('/api/admin/packages'),
  savePackage: (kind, p) => req(`/api/admin/packages/${kind}`, { method: 'POST', body: p }),
  deletePackage: (kind, id) => req(`/api/admin/packages/${kind}/${id}`, { method: 'DELETE' }),

  users: (page = 1, keyword = '') => req(`/api/admin/users?page=${page}&size=20&keyword=${encodeURIComponent(keyword)}`),
  adjust: (user_id, coins, remark) => req('/api/admin/users/adjust', { method: 'POST', body: { user_id, coins, remark } }),

  tenants: () => req('/api/admin/tenants'),
  createTenant: (t) => req('/api/admin/tenants', { method: 'POST', body: t }),

  orders: (page = 1, status = '') => req(`/api/admin/orders?page=${page}&size=20&status=${status}`),
  markPaid: (orderNo) => req(`/api/admin/orders/${orderNo}/mark-paid`, { method: 'POST' }),
}
