<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { t } from '../i18n'

const router = useRouter()
const step = ref(0)

const accounts = ref([])
const clips = ref([])
const selAccounts = ref([])
const selClips = ref([])

const rule = ref('ONE_TO_ONE')
const loopWhenShort = ref(false)
const maxRepeat = ref(1)
const name = ref('')
const caption = ref('{clip}\n#shortdrama #drama')
const privacy = ref('SELF_ONLY')
const scheduleMode = ref('now')
const scheduleAt = ref(null)
const disableComment = ref(false)
const disableDuet = ref(false)
const disableStitch = ref(false)

const preview = ref({ total: 0, per_account: [] })
const submitting = ref(false)

const rules = [
  { value: 'ONE_TO_ONE', key: 'ruleOneToOne' },
  { 'value': 'ALL_VIDEOS', key: 'ruleAllVideos' },
  { value: 'RANDOM_EVEN', key: 'ruleRandomEven' },
]

onMounted(async () => {
  try {
    const [a, c] = await Promise.all([api.accounts(), api.clips()])
    accounts.value = a.filter(x => x.status === 1)
    clips.value = c
  } catch (e) {
    ElMessage.error(e.message)
  }
})

const canNext = computed(() => {
  if (step.value === 0) return selAccounts.value.length > 0
  if (step.value === 1) return selClips.value.length > 0
  return caption.value.trim().length > 0
})

async function refreshPreview() {
  if (!selAccounts.value.length || !selClips.value.length) {
    preview.value = { total: 0, per_account: [] }
    return
  }
  try {
    preview.value = await api.previewTask({
      account_ids: selAccounts.value,
      clip_ids: selClips.value,
      rule: rule.value,
      loop_when_short: loopWhenShort.value,
      max_repeat: maxRepeat.value,
    })
  } catch { preview.value = { total: 0, per_account: [] } }
}

function next() {
  if (!canNext.value) return
  step.value++
  if (step.value === 2) refreshPreview()
}

async function submit() {
  if (!name.value.trim()) name.value = 'task-' + Date.now().toString(36)
  submitting.value = true
  try {
    const payload = {
      name: name.value,
      account_ids: selAccounts.value,
      clip_ids: selClips.value,
      rule: rule.value,
      loop_when_short: loopWhenShort.value,
      max_repeat: maxRepeat.value,
      title: caption.value,
      privacy_level: privacy.value,
      disable_comment: disableComment.value,
      disable_duet: disableDuet.value,
      disable_stitch: disableStitch.value,
    }
    if (scheduleMode.value === 'later' && scheduleAt.value) {
      payload.scheduled_at = new Date(scheduleAt.value).toISOString()
    }
    const r = await api.createTask(payload)
    ElMessage.success(t('previewTotal') + ': ' + r.total)
    router.replace('/tasks')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div>
    <h2>{{ t('navNewTask') }}</h2>
    <el-steps :active="step" finish-status="success" style="margin-bottom: 24px">
      <el-step :title="t('stepAccounts')" />
      <el-step :title="t('stepClips')" />
      <el-step :title="t('stepSettings')" />
    </el-steps>

    <!-- step 1: accounts -->
    <el-card v-if="step === 0">
      <template #header>{{ t('pickAccounts') }} ({{ selAccounts.length }} {{ t('selected') }})</template>
      <el-table :data="accounts" stripe @selection-change="s => selAccounts = s.map(x => x.id)">
        <el-table-column type="selection" width="50" />
        <el-table-column prop="display_name" min-width="180" />
        <el-table-column prop="open_id" min-width="220" show-overflow-tooltip />
        <el-table-column :label="t('status')" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? t('accOK') : t('accDisconnected') }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- step 2: clips -->
    <el-card v-if="step === 1">
      <template #header>{{ t('pickClips') }} ({{ selClips.length }} {{ t('selected') }})</template>
      <el-table :data="clips" stripe @selection-change="s => selClips = s.map(x => x.id)">
        <el-table-column type="selection" width="50" />
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="title" min-width="220" show-overflow-tooltip />
        <el-table-column prop="duration_sec" :label="t('duration')" width="90">
          <template #default="{ row }">{{ row.duration_sec }}s</template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- step 3: settings -->
    <div v-if="step === 2">
      <el-card>
        <el-form label-width="180px">
          <el-form-item :label="t('taskName')">
            <el-input v-model="name" style="width: 360px" />
          </el-form-item>
          <el-form-item :label="t('rule')">
            <el-radio-group v-model="rule" @change="refreshPreview">
              <el-radio v-for="r in rules" :key="r.value" :value="r.value">{{ t(r.key) }}</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="rule === 'ONE_TO_ONE'">
            <el-checkbox v-model="loopWhenShort" @change="refreshPreview">{{ t('loopWhenShort') }}</el-checkbox>
          </el-form-item>
          <el-form-item v-if="rule === 'RANDOM_EVEN'" :label="t('maxRepeat')">
            <el-input-number v-model="maxRepeat" :min="1" :max="20" @change="refreshPreview" />
          </el-form-item>
          <el-form-item :label="t('caption')">
            <el-input v-model="caption" type="textarea" :rows="4" style="max-width: 560px" />
            <div style="color: #999; font-size: 12px">{{ t('captionHint') }}</div>
          </el-form-item>
          <el-form-item :label="t('privacy')">
            <el-select v-model="privacy" style="width: 240px">
              <el-option value="SELF_ONLY" :label="t('stDelivered')" />
              <el-option value="MUTUAL_FOLLOW" label="Mutual follow" />
              <el-option value="FOLLOWER_OF_CREATOR" label="Followers" />
              <el-option value="PUBLIC_TO_EVERYONE" :label="t('stPublished')" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('scheduleAt')">
            <el-radio-group v-model="scheduleMode">
              <el-radio value="now">{{ t('scheduleNow') }}</el-radio>
              <el-radio value="later">{{ t('scheduleLater') }}</el-radio>
            </el-radio-group>
            <el-date-picker v-if="scheduleMode === 'later'" v-model="scheduleAt" type="datetime"
                            style="margin-left: 12px" />
          </el-form-item>
          <el-form-item :label="t('actions')">
            <el-checkbox v-model="disableComment" style="margin-right: 16px">{{ t('disableComment') }}</el-checkbox>
            <el-checkbox v-model="disableDuet" style="margin-right: 16px">{{ t('disableDuet') }}</el-checkbox>
            <el-checkbox v-model="disableStitch">{{ t('disableStitch') }}</el-checkbox>
          </el-form-item>
        </el-form>
      </el-card>
      <el-card style="margin-top: 16px">
        <el-statistic :title="t('previewTotal')" :value="preview.total" />
        <div v-if="preview.per_account && preview.per_account.length" style="color: #999; font-size: 13px; margin-top: 8px">
          {{ preview.per_account.map(p => p.count).join(' / ') }}
        </div>
      </el-card>
    </div>

    <div style="margin-top: 20px; display: flex; gap: 12px">
      <el-button v-if="step > 0" @click="step--">{{ t('prev') }}</el-button>
      <el-button v-if="step < 2" type="primary" :disabled="!canNext" @click="next">{{ t('next') }}</el-button>
      <el-button v-if="step === 2" type="primary" :loading="submitting"
                 :disabled="!canNext || preview.total === 0" @click="submit">{{ t('createTask') }}</el-button>
    </div>
  </div>
</template>
