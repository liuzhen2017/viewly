<script setup>
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { t } from '../i18n'

const accounts = ref(0)
const clips = ref(0)
const activeTasks = ref(0)
const delivered = ref(0)
const failed = ref(0)
const loaded = ref(false)

onMounted(async () => {
  try {
    const [accs, cls, tasks] = await Promise.all([api.accounts(), api.clips(), api.tasks()])
    accounts.value = accs.length
    clips.value = cls.length
    activeTasks.value = tasks.filter(x => x.status === 'running' || x.status === 'scheduled').length
    for (const task of tasks) {
      delivered.value += task.counts.delivered || 0
      delivered.value += task.counts.published || 0
      failed.value += task.counts.failed || 0
    }
  } catch { /* 401 handled in api.js */ }
  loaded.value = true
})
</script>

<template>
  <div>
    <h2>{{ t('dashHello') }}</h2>
    <el-row :gutter="16" v-if="loaded">
      <el-col :span="6"><el-card><el-statistic :title="t('dashAccounts')" :value="accounts" /></el-card></el-col>
      <el-col :span="6"><el-card><el-statistic :title="t('dashClips')" :value="clips" /></el-card></el-col>
      <el-col :span="6"><el-card><el-statistic :title="t('dashActive')" :value="activeTasks" /></el-card></el-col>
      <el-col :span="6"><el-card><el-statistic :title="t('dashDelivered')" :value="delivered" /></el-card></el-col>
    </el-row>
    <el-card style="margin-top: 16px">
      <el-statistic :title="t('dashFailed')" :value="failed" />
      <p style="color: #999; margin-top: 12px">{{ t('dashEmpty') }}</p>
    </el-card>
  </div>
</template>
