// Creator workspace (studio.likeviewly.com). End-user JWT auth (the same
// users as the H5 app), tenant resolved from the subdomain like everywhere
// else — 'studio' itself is reserved and maps to 'main'.
const RESERVED = new Set(['www', 'admin', 'api', 'cdn', 'studio'])

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

const TOKEN_KEY = 'viewly_studio_token_' + tenantSlug()
export const getToken = () => localStorage.getItem(TOKEN_KEY) || ''
export const setToken = (t) => localStorage.setItem(TOKEN_KEY, t)
export const clearToken = () => localStorage.removeItem(TOKEN_KEY)

function authHeaders() {
  const h = { 'X-Tenant-Slug': tenantSlug() }
  if (getToken()) h.Authorization = 'Bearer ' + getToken()
  return h
}

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

// ---- chunked multipart upload (same machinery as admin, user-scoped) ----
const CHUNK_SIZE = 16 * 1024 * 1024
const CHUNK_CONCURRENCY = 6

function post(path, body, direct) {
  const base = (direct && !location.hostname.endsWith('.localhost')) ? 'https://upload.likeviewly.com' : ''
  return req(base + path, { method: 'POST', body })
}

export async function uploadLargeFile(file, onProgress) {
  const total = Math.ceil(file.size / CHUNK_SIZE)
  let base = (location.hostname.endsWith('.localhost')) ? '' : 'https://upload.likeviewly.com'
  let init
  try {
    init = await post('/api/user/uploads/mp/init', { filename: file.name }, true)
  } catch (e) {
    if (!base) throw e
    base = ''
    init = await post('/api/user/uploads/mp/init', { filename: file.name }, false)
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
        const r = await fetchT(base + '/api/user/uploads/mp/chunk?upload_id=' + uploadId + '&part=' + partNo, {
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
    post('/api/user/uploads/mp/abort', { upload_id: uploadId }, true).catch(() => {})
    throw e
  }
  parts.sort((a, b) => a.part - b.part)
  const done = await fetchT(base + '/api/user/uploads/mp/complete', {
    method: 'POST', headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ upload_id: uploadId, parts }),
  }, 60000).then(r => r.json())
  if (done.code !== 0) throw new Error(done.msg)
  return done.data
}

export async function uploadSmallFile(file, onProgress) {
  // relay upload (small files): browser -> API -> S3
  const tryDirect = async (direct) => new Promise((resolve, reject) => {
    const fd = new FormData()
    fd.append('file', file)
    const base = (direct && !location.hostname.endsWith('.localhost')) ? 'https://upload.likeviewly.com' : ''
    const xhr = new XMLHttpRequest()
    xhr.open('POST', base + '/api/user/uploads')
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
    xhr.send(fd)
  })
  try { return await tryDirect(true) } catch { return tryDirect(false) }
}

export const api = {
  // auth (same user accounts as the H5 app)
  register: (email, password) => req('/api/auth/register', { method: 'POST', body: { email, password } }),
  login: (email, password) => req('/api/auth/login', { method: 'POST', body: { email, password } }),
  me: () => req('/api/user/me'),

  // drama catalog (public endpoints)
  dramas: (page = 1, keyword = '') => req(`/api/dramas?page=${page}&size=24&keyword=${encodeURIComponent(keyword)}`),
  dramaDetail: (id) => req(`/api/dramas/${id}`),

  // TikTok accounts + authorization links
  accounts: () => req('/api/tk/accounts'),
  deleteAccount: (id) => req(`/api/tk/accounts/${id}`, { method: 'DELETE' }),
  connectURL: () => req('/api/tk/connect-url', { method: 'POST', body: {} }),
  createAuthLink: (title) => req('/api/tk/auth-links', { method: 'POST', body: { title } }),
  authLinks: () => req('/api/tk/auth-links'),
  revokeAuthLink: (id) => req(`/api/tk/auth-links/${id}`, { method: 'DELETE' }),

  // clips + slicing
  clips: () => req('/api/tk/clips'),
  saveClip: (c) => req('/api/tk/clips', { method: 'POST', body: c }),
  deleteClip: (id) => req(`/api/tk/clips/${id}`, { method: 'DELETE' }),
  createSliceJob: (episode_id, seg_sec, max_count) => req('/api/tk/slice-jobs', { method: 'POST', body: { episode_id, seg_sec, max_count } }),
  sliceJobs: () => req('/api/tk/slice-jobs'),

  // batch tasks
  previewTask: (payload) => req('/api/tk/tasks/preview', { method: 'POST', body: payload }),
  createTask: (payload) => req('/api/tk/tasks', { method: 'POST', body: payload }),
  tasks: () => req('/api/tk/tasks'),
  taskDetail: (id) => req(`/api/tk/tasks/${id}`),
  cancelTask: (id) => req(`/api/tk/tasks/${id}/cancel`, { method: 'POST', body: {} }),
  checkPost: (id) => req(`/api/tk/posts/${id}/check`, { method: 'POST', body: {} }),
  posts: (status = '', page = 1) => req(`/api/tk/posts?page=${page}&size=50${status ? '&status=' + status : ''}`),
}
