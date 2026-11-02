<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { t } from '../i18n'

const tasks = ref([])
const expanded = ref(null) // task id whose posts are shown
const posts = ref([])
let timer = null

const taskStatusType = { scheduled: 'warning', running: 'primary', done: 'success', canceled: 'info' }
const taskStatusLabel = { scheduled: 'tsScheduled', running: 'tsRunning', done: 'tsDone', canceled: 'tsCanceled' }
const postStatusType = {
  scheduled: 'info', queued: 'info', uploading: 'warning', processing: 'warning',
  delivered: 'success', published: 'success', failed: 'danger', canceled: 'info',
}
const postStatusLabel = {
  scheduled: 'stScheduled', queued: 'stQueued', uploading: 'stUploading', processing: 'stProcessing',
  delivered: 'stDelivered', published: 'stPublished', failed: 'stFailed', canceled: 'stCanceled',
}

const countKeys = ['delivered', 'published', 'failed', 'canceled', 'queued', 'uploading', 'processing', 'scheduled']

const active = () => tasks.value.some(x => x.status === 'running' || x.status === 'scheduled')

async function load() {
  try {
    tasks.value = await api.tasks()
    if (expanded.value) {
      const still = tasks.value.find(x => x.id === expanded.value)
      if (still) await loadPosts(still.id)
    }
  } catch (e) {
    ElMessage.error(e.message)
  }
}
onMounted(() => {
  load()
  timer = setInterval(() => { if (active()) load() }, 5000)
})
onUnmounted(() => clearInterval(timer))

async function loadPosts(id) {
  expanded.value = id
  try {
    const r = await api.taskDetail(id)
    posts.value = r.posts || []
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function cancelTask(task) {
  try {
    await ElMessageBox.confirm(t('cancelTaskConfirm'), t('cancelTask'), { type: 'warning' })
    await api.cancelTask(task.id)
    ElMessage.success(t('stCanceled'))
    load()
  } catch { /* cancelled */ }
}

async function check(row) {
  try {
    await api.checkPost(row.id)
    ElMessage.success(t('checked'))
    if (expanded.value) loadPosts(expanded.value)
  } catch (e) {
    ElMessage.error(e.message)
  }
}
</script>

<template>
  <div>
    <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px">
      <h2 style="margin: 0">{{ t('navTasks') }}</h2>
      <el-button @click="load">{{ t('refresh') }}</el-button>
    </div>

    <el-table :data="tasks" stripe row-key="id">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" :label="t('taskName')" min-width="150" show-overflow-tooltip />
      <el-table-column prop="rule" :label="t('taskRule')" width="130" />
      <el-table-column prop="total" :label="t('taskTotal')" width="80" />
      <el-table-column :label="t('taskStatus')" width="110">
        <template #default="{ row }">
          <el-tag size="small" :type="taskStatusType[row.status]">{{ t(taskStatusLabel[row.status] || row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('status')" min-width="260">
        <template #default="{ row }">
          <template v-for="k in countKeys" :key="k">
            <el-tag v-if="row.counts && row.counts[k]" size="small" :type="postStatusType[k]"
                    style="margin-right: 4px">
              {{ t(postStatusLabel[k]) }} {{ row.counts[k] }}
            </el-tag>
          </template>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" :label="t('createdAt')" width="170">
        <template #default="{ row }">{{ (row.created_at || '').replace('T', ' ').slice(0, 16) }}</template>
      </el-table-column>
      <el-table-column :label="t('actions')" width="190">
        <template #default="{ row }">
          <el-button size="small" @click="loadPosts(row.id)">{{ t('viewPosts') }}</el-button>
          <el-button v-if="row.status === 'running' || row.status === 'scheduled'" size="small" type="danger"
                     @click="cancelTask(row)">{{ t('cancel') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-card v-if="expanded" style="margin-top: 16px">
      <template #header>
        {{ t('navTasks') }} #{{ expanded }} — {{ posts.length }} {{ t('taskTotal') }}
      </template>
      <el-table :data="posts" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="account_name" :label="t('account')" min-width="130" show-overflow-tooltip />
        <el-table-column prop="clip_title" :label="t('clip')" min-width="160" show-overflow-tooltip />
        <el-table-column :label="t('status')" width="140">
          <template #default="{ row }">
            <el-tag size="small" :type="postStatusType[row.status]">{{ t(postStatusLabel[row.status] || row.status) }}</el-tag>
            <el-tag v-if="row.status === 'delivered' && !row.checked_at" size="small" type="warning" effect="plain">
              {{ t('stPendingCheck') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error" min-width="160" show-overflow-tooltip />
        <el-table-column :label="t('actions')" width="120">
          <template #default="{ row }">
            <el-button v-if="row.status === 'delivered' && !row.checked_at" size="small" type="primary"
                       @click="check(row)">{{ t('check') }}</el-button>
            <span v-else-if="row.checked_at" style="color: #67c23a; font-size: 13px">{{ t('checked') }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>
