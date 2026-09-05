// Multi-tenant admin: X-Tenant-Slug selects the tenant site being managed
// (subdomain in production, localStorage override for local testing).
// Tokens and the admin role are stored per tenant.
const RESERVED = new Set(['www', 'admin', 'api', 'cdn'])

export function tenantSlug() {
  const saved = localStorage.getItem('viewly_tenant')
  if (saved) return saved
  const host = location.hostname
  if (host.endsWith('.localhost')) {
    const s = host.slice(0, -'.localhost'.length)
    return RESERVED.has(s) ? 'main' : s
  }
  const parts = host.split('.')
  if (parts.length <= 2) return 'main'
  return RESERVED.has(parts[0]) ? 'main' : parts[0]
}

const TOKEN_KEY = 'viewly_admin_token_' + tenantSlug()
const ROLE_KEY = 'viewly_admin_role_' + tenantSlug()
export const getToken = () => localStorage.getItem(TOKEN_KEY) || ''
export const setToken = (t) => localStorage.setItem(TOKEN_KEY, t)
export const getRole = () => localStorage.getItem(ROLE_KEY) || ''

function authHeaders() {
  const h = { 'X-Tenant-Slug': tenantSlug() }
  if (getToken()) h.Authorization = 'Bearer ' + getToken()
  return h
}
export const setRole = (r) => localStorage.setItem(ROLE_KEY, r)
export const clearToken = () => { localStorage.removeItem(TOKEN_KEY); localStorage.removeItem(ROLE_KEY) }

// fetch with a hard timeout: without this a silently-dropped connection
// (proxy node cycling, CF stall) hangs forever and freezes the whole batch.
function fetchT(url, opts = {}, ms = 30000) {
  const ctl = new AbortController()
  const timer = setTimeout(() => ctl.abort(), ms)
  return fetch(url, { ...opts, signal: ctl.signal })
    .finally(() => clearTimeout(timer))
}

export async function req(path, { method = 'GET', body } = {}) {
  const headers = { 'Content-Type': 'application/json', 'X-Tenant-Slug': tenantSlug() }
  if (getToken()) headers.Authorization = 'Bearer ' + getToken()
  const res = await fetchT(path, {
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

// uploadWithFallback: try direct host, fall back to same-origin on failure
export async function uploadWithFallback(file, onProgress) {
  try {
    return await admin.uploadFile(file, onProgress, true)
  } catch (e) {
    if (/direct route/.test(e.message)) return admin.uploadFile(file, onProgress, false)
    throw e
  }
}

// ---- chunked multipart upload for large files ----
// 16MB chunks, 3 concurrent, per-chunk retry. Every request stays small so
// proxy body limits (CF free = 100MB) can never kill an upload again.
const CHUNK_SIZE = 16 * 1024 * 1024
const CHUNK_CONCURRENCY = 6

function post(path, body, direct) {
  const base = (direct && !location.hostname.endsWith('.localhost')) ? 'https://upload.likeviewly.com' : ''
  return req(base + path, { method: 'POST', body })
}

export async function uploadLargeFile(file, onProgress) {
  const total = Math.ceil(file.size / CHUNK_SIZE)
  // pick one route for the whole transfer: direct first, same-origin fallback
  let base = (location.hostname.endsWith('.localhost')) ? '' : 'https://upload.likeviewly.com'
  let init
  try {
    init = await post('/api/admin/uploads/mp/init', { filename: file.name }, true)
  } catch (e) {
    if (!base) throw e
    base = ''
    init = await post('/api/admin/uploads/mp/init', { filename: file.name }, false)
  }
  const uploadId = init.upload_id
  const parts = []
  let doneBytes = 0
  let cursor = 1

  const uploadChunk = async partNo => {
    const blob = file.slice((partNo - 1) * CHUNK_SIZE, Math.min(partNo * CHUNK_SIZE, file.size))
    for (let attempt = 0; attempt < 4; attempt++) {
      try {
        const fd = new FormData()
        fd.append('chunk', blob)
        // 180s hard timeout: a 16MB chunk needs ~107s even at 0.15MB/s, so this
        // only fires on a truly dead connection — then the retry takes over.
        const r = await fetchT(base + '/api/admin/uploads/mp/chunk?upload_id=' + uploadId + '&part=' + partNo, {
          method: 'POST', body: fd, headers: authHeaders(),
        }, 180000)
        const d = await r.json()
        if (d.code !== 0) throw new Error(d.msg)
        parts.push({ part: partNo, etag: d.data.etag })
        doneBytes += blob.size
        if (onProgress) onProgress(Math.round(doneBytes / file.size * 100))
        return
      } catch (e) {
        if (attempt === 3) throw new Error('chunk ' + partNo + ' failed: ' + (e.name === 'AbortError' ? 'timeout' : e.message))
        await new Promise(r2 => setTimeout(r2, 1000 * (attempt + 1)))
      }
    }
  }

  const workers = Array.from({ length: CHUNK_CONCURRENCY }, async () => {
    while (true) {
      const my = cursor++
      if (my > total) break
      await uploadChunk(my)
    }
  })
  try {
    await Promise.all(workers)
  } catch (e) {
    post('/api/admin/uploads/mp/abort', { upload_id: uploadId }, true).catch(() => {})
    throw e
  }
  parts.sort((a, b) => a.part - b.part)
  const done = await fetchT(base + '/api/admin/uploads/mp/complete', {
    method: 'POST', headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ upload_id: uploadId, parts }),
  }, 60000).then(r => r.json())
  if (done.code !== 0) throw new Error(done.msg)
  return done.data
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
  batchPrice: (ids, priceCoins) => req('/api/admin/episodes/batch-price', { method: 'PUT', body: { ids, price_coins: priceCoins } }),

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

  adSettings: () => req('/api/admin/ad-settings'),
  presign: (filename, content_type) => req('/api/admin/uploads/presign', { method: 'POST', body: { filename, content_type } }),

// relay upload (no CORS issues): browser -> API -> S3, with progress callback.
  // Prefers the direct upload host (bypasses the CF proxy for speed);
  // falls back to the same-origin route automatically if it fails.
  uploadFile: (file, onProgress, direct = true) => new Promise((resolve, reject) => {
    const fd = new FormData()
    fd.append('file', file)
    const base = (direct && !location.hostname.endsWith('.localhost')) ? 'https://upload.likeviewly.com' : ''
    const xhr = new XMLHttpRequest()
    xhr.open('POST', base + '/api/admin/uploads')
    const token = getToken()
    if (token) xhr.setRequestHeader('Authorization', 'Bearer ' + token)
    xhr.setRequestHeader('X-Tenant-Slug', tenantSlug())
    xhr.upload.onprogress = e => { if (e.lengthComputable && onProgress) onProgress(Math.round(e.loaded / e.total * 100)) }
    xhr.onload = () => {
      try {
        const d = JSON.parse(xhr.responseText)
        if (d.code === 0) resolve(d.data)
        else reject(new Error(d.msg || 'upload failed'))
      } catch { reject(new Error('upload failed: HTTP ' + xhr.status)) }
    }
    xhr.onerror = () => reject(new Error(direct ? 'direct route unreachable' : 'network error'))
    xhr.timeout = 120000
    xhr.ontimeout = () => reject(new Error(direct ? 'direct route timeout' : 'upload timeout'))
    xhr.send(fd)
  }),
  saveAdSettings: (s) => req('/api/admin/ad-settings', { method: 'PUT', body: s }),

  orders: (page = 1, status = '') => req(`/api/admin/orders?page=${page}&size=20&status=${status}`),
  markPaid: (orderNo) => req(`/api/admin/orders/${orderNo}/mark-paid`, { method: 'POST' }),
}
