<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { t } from '../i18n'

const dramas = ref([])
const page = ref(1)
const keyword = ref('')
const total = ref(0)
const loading = ref(false)

const drawer = ref(false)
const current = ref(null)
const episodes = ref([])

const sliceDlg = ref(false)
const sliceEp = ref(null)
const segSec = ref(60)
const maxCount = ref(10)
const slicing = ref(false)

async function load() {
  loading.value = true
  try {
    const r = await api.dramas(page.value, keyword.value)
    dramas.value = r.list || r.dramas || r
    total.value = r.total || dramas.value.length
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function openDrama(d) {
  current.value = d
  episodes.value = []
  drawer.value = true
  try {
    const r = await api.dramaDetail(d.id)
    episodes.value = r.episodes || []
  } catch (e) {
    ElMessage.error(e.message)
  }
}

function openSlice(ep) {
  sliceEp.value = ep
  sliceDlg.value = true
}

async function startSlice() {
  slicing.value = true
  try {
    await api.createSliceJob(sliceEp.value.id, segSec.value, maxCount.value)
    ElMessage.success(t('sliceStarted'))
    sliceDlg.value = false
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    slicing.value = false
  }
}
</script>

<template>
  <div>
    <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px">
      <h2 style="margin: 0">{{ t('dramasTitle') }}</h2>
      <el-input v-model="keyword" :placeholder="t('search')" style="width: 240px" clearable
                @keyup.enter="page = 1; load()" />
      <el-button @click="page = 1; load()">{{ t('search') }}</el-button>
    </div>

    <el-row :gutter="16" v-loading="loading">
      <el-col v-for="d in dramas" :key="d.id" :span="6" style="margin-bottom: 16px">
        <el-card shadow="hover" style="cursor: pointer" @click="openDrama(d)">
          <el-image :src="d.cover" fit="cover" style="width: 100%; height: 200px; border-radius: 8px" />
          <div style="margin-top: 8px; font-weight: 600">{{ d.title }}</div>
          <div style="color: #999; font-size: 13px; margin-top: 4px">
            {{ t('episodes') }}: {{ d.episodes }}
          </div>
        </el-card>
      </el-col>
    </el-row>
    <el-pagination v-model:current-page="page" :page-size="24" :total="total" layout="prev, pager, next"
                   @current-change="load" />

    <el-drawer v-model="drawer" :title="current ? current.title : ''" size="50%">
      <el-table :data="episodes" stripe>
        <el-table-column prop="ep_index" label="#" width="60" />
        <el-table-column prop="title" :label="t('episodes')" min-width="160" show-overflow-tooltip />
        <el-table-column prop="duration_sec" :label="t('duration')" width="100">
          <template #default="{ row }">{{ Math.floor(row.duration_sec / 60) }}m{{ row.duration_sec % 60 }}s</template>
        </el-table-column>
        <el-table-column :label="t('actions')" width="120">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openSlice(row)">{{ t('slice') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <el-dialog v-model="sliceDlg" :title="t('slice')" width="420px">
      <p style="color: #999; font-size: 13px">{{ t('sliceHint') }}</p>
      <el-form label-width="140px">
        <el-form-item :label="t('segSec')">
          <el-input-number v-model="segSec" :min="10" :max="600" />
        </el-form-item>
        <el-form-item :label="t('maxCount')">
          <el-input-number v-model="maxCount" :min="1" :max="50" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="sliceDlg = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" :loading="slicing" @click="startSlice">{{ t('startSlice') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>
