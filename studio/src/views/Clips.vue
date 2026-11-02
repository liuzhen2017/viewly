<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, uploadLargeFile, uploadSmallFile } from '../api'
import { t } from '../i18n'

const clips = ref([])
const jobs = ref([])
const uploading = ref(false)
const uploadPct = ref(0)
const uploadName = ref('')
let timer = null

const jobStatusType = { queued: 'info', running: 'warning', done: 'success', failed: 'danger' }

async function load() {
  try {
    const [c, j] = await Promise.all([api.clips(), api.sliceJobs()])
    clips.value = c
    jobs.value = j
  } catch (e) {
    ElMessage.error(e.message)
  }
}
onMounted(() => {
  load()
  // slice jobs run server-side; poll while anything is in flight
  timer = setInterval(() => {
    if (jobs.value.some(x => x.status === 'queued' || x.status === 'running')) load()
  }, 5000)
})
onUnmounted(() => clearInterval(timer))

function fmtSize(n) {
  if (!n) return '0'
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(1) + 'G'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + 'M'
  return (n / 1024).toFixed(0) + 'K'
}

function readDuration(file) {
  return new Promise(resolve => {
    const url = URL.createObjectURL(file)
    const v = document.createElement('video')
    v.preload = 'metadata'
    v.onloadedmetadata = () => { URL.revokeObjectURL(url); resolve(Math.round(v.duration) || 0) }
    v.onerror = () => { URL.revokeObjectURL(url); resolve(0) }
    v.src = url
  })
}

async function onPicked(ev) {
  const files = [...ev.target.files]
  ev.target.value = ''
  uploading.value = true
  let done = 0
  try {
    for (const f of files) {
      uploadName.value = `${++done}/${files.length} ${f.name}`
      const up = f.size > 20 * 1024 * 1024 ? uploadLargeFile : uploadSmallFile
      const r = await up(f, p => { uploadPct.value = p })
      const duration = await readDuration(f)
      await api.saveClip({
        title: f.name.replace(/\.[^.]+$/, ''),
        video_url: r.cdn_url,
        size_bytes: f.size,
        duration_sec: duration,
      })
    }
    ElMessage.success(t('uploaded'))
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    uploading.value = false
    uploadPct.value = 0
  }
}

async function del(c) {
  try {
    await ElMessageBox.confirm(c.title, t('del'), { type: 'warning' })
    await api.deleteClip(c.id)
    load()
  } catch { /* cancelled */ }
}
</script>

<template>
  <div>
    <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px">
      <h2 style="margin: 0">{{ t('clipsTitle') }}</h2>
      <label class="el-button el-button--primary" style="position: relative; overflow: hidden">
        {{ t('upload') }}
        <input type="file" accept="video/*" multiple style="position: absolute; inset: 0; opacity: 0"
               @change="onPicked" />
      </label>
      <el-button @click="load">{{ t('refresh') }}</el-button>
      <el-progress v-if="uploading" :percentage="uploadPct" style="width: 240px" />
      <span v-if="uploading" style="font-size: 13px; color: #999">{{ uploadName }}</span>
    </div>

    <el-table :data="clips" stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="title" min-width="200" show-overflow-tooltip />
      <el-table-column prop="duration_sec" :label="t('duration')" width="90">
        <template #default="{ row }">{{ row.duration_sec }}s</template>
      </el-table-column>
      <el-table-column prop="size_bytes" :label="t('size')" width="90">
        <template #default="{ row }">{{ fmtSize(row.size_bytes) }}</template>
      </el-table-column>
      <el-table-column prop="created_at" :label="t('createdAt')" width="170">
        <template #default="{ row }">{{ (row.created_at || '').replace('T', ' ').slice(0, 16) }}</template>
      </el-table-column>
      <el-table-column :label="t('actions')" width="90">
        <template #default="{ row }">
          <el-button size="small" type="danger" @click="del(row)">{{ t('del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <h3 style="margin-top: 28px">{{ t('sliceJobs') }}</h3>
    <el-table :data="jobs" stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="drama_title" min-width="150" show-overflow-tooltip />
      <el-table-column prop="episode_title" min-width="150" show-overflow-tooltip />
      <el-table-column :label="t('segSec')" width="90">
        <template #default="{ row }">{{ row.seg_sec }}s</template>
      </el-table-column>
      <el-table-column :label="t('clipCount')" width="90">
        <template #default="{ row }">{{ row.status === 'done' ? row.clip_count : '—' }}</template>
      </el-table-column>
      <el-table-column :label="t('status')" width="110">
        <template #default="{ row }">
          <el-tag size="small" :type="jobStatusType[row.status]">
            {{ t({ queued: 'jobQueued', running: 'jobRunning', done: 'jobDone', failed: 'jobFailed' }[row.status] || row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="error" min-width="180" show-overflow-tooltip />
    </el-table>
  </div>
</template>
