<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { t } from '../i18n'

const accounts = ref([])
const links = ref([])

async function load() {
  try {
    const [a, l] = await Promise.all([api.accounts(), api.authLinks()])
    accounts.value = a
    links.value = l
  } catch (e) {
    ElMessage.error(e.message)
  }
}
onMounted(load)

async function connect() {
  try {
    const r = await api.connectURL()
    window.open(r.url, '_blank')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function del(a) {
  try {
    await ElMessageBox.confirm(a.display_name || a.open_id, t('del'), { type: 'warning' })
    await api.deleteAccount(a.id)
    load()
  } catch { /* cancelled */ }
}

async function createLink() {
  try {
    const { value } = await ElMessageBox.prompt(t('linkTitle'), t('createLink'), { inputValue: '' })
    await api.createAuthLink(value || '')
    load()
  } catch { /* cancelled */ }
}

async function copy(text) {
  await navigator.clipboard.writeText(text)
  ElMessage.success(t('copied'))
}

async function revoke(l) {
  try {
    await ElMessageBox.confirm(l.title, t('revoke'), { type: 'warning' })
    await api.revokeAuthLink(l.id)
    load()
  } catch { /* cancelled */ }
}

function fmtExpiry(sec) {
  if (sec <= 0) return t('expired')
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  return d > 0 ? `${d}d ${h}h` : `${h}h`
}
</script>

<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; align-items: center; gap: 12px">
          <span style="font-weight: 700">{{ t('navAccounts') }}</span>
          <el-button type="primary" @click="connect">{{ t('connect') }}</el-button>
          <el-button @click="load">{{ t('refresh') }}</el-button>
          <span style="color: #999; font-size: 13px">{{ t('connectTip') }}</span>
        </div>
      </template>
      <el-table :data="accounts" stripe>
        <el-table-column :label="t('displayName')" min-width="200">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 8px">
              <el-avatar :src="row.avatar" :size="28" />
              {{ row.display_name }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="open_id" :label="t('openID')" min-width="220" show-overflow-tooltip />
        <el-table-column :label="t('status')" width="140">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? t('accOK') : t('accDisconnected') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('actions')" width="90">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="del(row)">{{ t('del') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card style="margin-top: 16px">
      <template #header>
        <div style="display: flex; align-items: center; gap: 12px">
          <span style="font-weight: 700">{{ t('authLinks') }}</span>
          <el-button type="primary" @click="createLink">{{ t('createLink') }}</el-button>
          <span style="color: #999; font-size: 13px">{{ t('linkHint') }}</span>
        </div>
      </template>
      <el-table :data="links" stripe>
        <el-table-column prop="title" min-width="140" show-overflow-tooltip />
        <el-table-column :label="t('useCount')" width="90">
          <template #default="{ row }">{{ row.use_count }}</template>
        </el-table-column>
        <el-table-column :label="t('usedBy')" min-width="140">
          <template #default="{ row }">{{ row.used_by_name || '—' }}</template>
        </el-table-column>
        <el-table-column :label="t('expires')" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.expired ? 'info' : 'success'">{{ fmtExpiry(row.expires_in_sec) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column min-width="220">
          <template #default="{ row }">
            <el-button size="small" @click="copy(row.url)">{{ t('copy') }}</el-button>
            <el-button size="small" type="danger" :disabled="row.expired" @click="revoke(row)">{{ t('revoke') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>
