<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { admin, uploadWithFallback, uploadLargeFile } from '../api'
import { t } from '../i18n'
import { useRoute } from 'vue-router'

const route = useRoute()
const accs = ref([])
const clips = ref([])
const posts = ref([])
const selAccs = ref([])
const selClips = ref([])
const caption = ref('Full episode on FoxDrama 🔥 {clip} https://www.likeviewly.com #shortdrama #fox')
const privacy = ref('SELF_ONLY')
const uploading = ref(0)
const uploadPct = ref(0)
const publishBusy = ref(false)
const clipInput = ref(null)

const VIDEO_EXT = /\.(mp4|mov|webm)$/i

function readDuration(file) {
  return new Promise(resolve => {
    const url = URL.createObjectURL(file)
    const v = document.createElement('video')
    v.preload = 'metadata'
    v.onloadedmetadata = () => { const d = Math.round(v.duration); URL.revokeObjectURL(url); resolve(d || 0) }
    v.onerror = () => { URL.revokeObjectURL(url); resolve(0) }
    v.src = url
  })
}

async function load() {
  try {
    accs.value = await admin.ttAccounts()
    clips.value = await admin.ttClips()
    posts.value = await admin.ttPosts()
  } catch (e) {
    ElMessage.error(e.message)
  }
}
onMounted(async () => {
  await load()
  if (route.query.connected) ElMessage.success('TikTok account connected ✓')
  if (route.query.err) ElMessage.error('Connect failed: ' + route.query.err)
})

async function connect() {
  try {
    const r = await admin.ttConnectURL()
    window.open(r.url, '_blank')
  } catch (e) { ElMessage.error(e.message) }
}

async function removeAcc(row) {
  await ElMessageBox.confirm(row.display_name || row.open_id, t('warning'), { type: 'warning' })
  await admin.ttDeleteAccount(row.id)
  load()
}

function pickClips() { clipInput.value && clipInput.value.click() }

async function onClipsPicked(e) {
  const files = [...(e.target.files || [])].filter(f => VIDEO_EXT.test(f.name))
  e.target.value = ''
  uploading.value = files.length
  let okc = 0, fail = 0
  for (const f of files) {
    try {
      const uploader = f.size > 20 * 1024 * 1024 ? uploadLargeFile : uploadWithFallback
      const [r, dur] = await Promise.all([uploader(f, pct => { uploadPct.value = pct }), readDuration(f)])
      await admin.ttSaveClip({ title: f.name.replace(VIDEO_EXT, ''), video_url: r.cdn_url, size_bytes: f.size, duration_sec: dur })
      okc++
    } catch (err) { fail++; ElMessage.error(f.name + ': ' + err.message) }
    uploading.value--
    uploadPct.value = 0
  }
  if (okc) ElMessage.success(okc + ' clip(s) saved')
  load()
}

async function removeClip(row) {
  await ElMessageBox.confirm(row.title, t('warning'), { type: 'warning' })
  await admin.ttDeleteClip(row.id)
  load()
}

async function publish() {
  if (!selAccs.value.length || !selClips.value.length) {
    ElMessage.warning(t('ttNeedSel'))
    return
  }
  publishBusy.value = true
  try {
    const r = await admin.ttPublish({
      account_ids: selAccs.value, clip_ids: selClips.value,
      title: caption.value, privacy_level: privacy.value,
    })
    ElMessage.success(r.queued + t('ttQueued'))
    await load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally { publishBusy.value = false }
}

function fmtMB(n) { return n ? (n / 1024 / 1024).toFixed(1) + ' MB' : '' }
function statusType(s) {
  return { published: 'success', failed: 'danger', uploading: 'warning', processing: 'warning', queued: 'info' }[s] || 'info'
}
function statusText(s) {
  return { queued: t('ttStatusQueued'), uploading: t('ttStatusUploading'), processing: t('ttStatusProcessing'), published: t('ttStatusPublished'), failed: t('ttStatusFailed') }[s] || s
}
</script>

<template>
  <el-card style="margin-bottom:14px">
    <template #header><b>{{ t('ttAccounts') }}</b></template>
    <p style="color:#888;font-size:12px;margin:0 0 10px">{{ t('ttConnectHint') }}</p>
    <el-button type="primary" @click="connect">➕ {{ t('ttConnect') }}</el-button>
    <el-table v-if="accs.length" :data="accs" size="small" style="margin-top:12px" @selection-change="selAccs = $event.map(x => x.id)">
      <el-table-column type="selection" width="42" />
      <el-table-column :label="t('title')" min-width="180">
        <template #default="{ row }">
          <span v-if="row.avatar"><img :src="row.avatar" style="width:22px;height:22px;border-radius:50%;vertical-align:-6px;margin-right:6px" /></span>
          {{ row.display_name || '(no name)' }}
        </template>
      </el-table-column>
      <el-table-column prop="open_id" label="Open ID" min-width="200" show-overflow-tooltip />
      <el-table-column :label="t('status')" width="120">
        <template #default="{ row }">
          <el-tag v-if="row.status === 1" size="small" type="success">OK</el-tag>
          <el-tag v-else size="small" type="danger">{{ t('ttStatusDisconnected') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('actions')" width="90">
        <template #default="{ row }">
          <el-button size="small" type="danger" @click="removeAcc(row)">{{ t('del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-else :description="t('ttNoAccounts')" :image-size="48" style="padding:18px 0" />
  </el-card>

  <el-card style="margin-bottom:14px">
    <template #header><b>{{ t('ttClips') }}</b></template>
    <div style="display:flex;gap:10px;align-items:center">
      <el-button type="success" :disabled="uploading > 0" @click="pickClips">
        {{ uploading > 0 ? t('uploading') + '… ' + uploadPct + '%' : '⬆ ' + t('ttUploadClips') }}
      </el-button>
      <input ref="clipInput" type="file" hidden multiple accept="video/*" @change="onClipsPicked" />
    </div>
    <el-table v-if="clips.length" :data="clips" size="small" style="margin-top:12px" @selection-change="selClips = $event.map(x => x.id)">
      <el-table-column type="selection" width="42" />
      <el-table-column prop="title" :label="t('epTitle')" min-width="180" show-overflow-tooltip />
      <el-table-column :label="t('duration')" width="90">
        <template #default="{ row }">{{ row.duration_sec }}s</template>
      </el-table-column>
      <el-table-column label="Size" width="100">
        <template #default="{ row }">{{ fmtMB(row.size_bytes) }}</template>
      </el-table-column>
      <el-table-column prop="created_at" label="Created" width="160">
        <template #default="{ row }">{{ (row.created_at || '').slice(0, 19) }}</template>
      </el-table-column>
      <el-table-column :label="t('actions')" width="90">
        <template #default="{ row }">
          <el-button size="small" type="danger" @click="removeClip(row)">{{ t('del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-else :description="t('ttNoClips')" :image-size="48" style="padding:18px 0" />
  </el-card>

  <el-card style="margin-bottom:14px">
    <template #header><b>{{ t('ttPublish') }}</b></template>
    <el-form label-width="130px">
      <el-form-item :label="t('ttCaptionTpl')">
        <el-input v-model="caption" type="textarea" :rows="3" :placeholder="t('ttCaptionPh')" />
      </el-form-item>
      <el-form-item :label="t('ttPrivacy')">
        <el-select v-model="privacy" style="width:280px">
          <el-option label="SELF_ONLY (draft / 审核前模式)" value="SELF_ONLY" />
          <el-option label="MUTUAL_FOLLOW" value="MUTUAL_FOLLOW" />
          <el-option label="FOLLOWER_OF_CREATOR" value="FOLLOWER_OF_CREATOR" />
          <el-option label="PUBLIC_TO_EVERYONE (需要应用过审)" value="PUBLIC_TO_EVERYONE" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" size="large" :loading="publishBusy" @click="publish">
          🚀 {{ t('ttPublishBtn').replace('{n}', selAccs.length * selClips.length) }}
        </el-button>
        <span style="margin-left:10px;color:#888;font-size:12px">{{ selAccs.length }} × {{ selClips.length }}</span>
      </el-form-item>
    </el-form>
  </el-card>

  <el-card>
    <template #header>
      <div style="display:flex;justify-content:space-between;align-items:center">
        <b>{{ t('ttPosts') }}</b>
        <el-button size="small" @click="load">{{ t('ttRefresh') }}</el-button>
      </div>
    </template>
    <el-table :data="posts" size="small">
      <el-table-column prop="id" label="#" width="60" />
      <el-table-column prop="account_name" :label="t('title')" min-width="130" show-overflow-tooltip />
      <el-table-column prop="clip_title" :label="t('ttClips')" min-width="150" show-overflow-tooltip />
      <el-table-column :label="t('status')" width="130">
        <template #default="{ row }">
          <el-tag size="small" :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="error" label="Error" min-width="180" show-overflow-tooltip />
      <el-table-column prop="created_at" label="Created" width="160">
        <template #default="{ row }">{{ (row.created_at || '').slice(0, 19) }}</template>
      </el-table-column>
    </el-table>
  </el-card>
</template>
