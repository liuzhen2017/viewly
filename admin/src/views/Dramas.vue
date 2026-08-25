<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { admin } from '../api'
import { t } from '../i18n'

const list = ref([])
const total = ref(0)
const page = ref(1)
const keyword = ref('')
const cats = ref([])
const dialog = ref(false)
const saving = ref(false)
const form = ref({})
const epDialog = ref(false)
const epDrama = ref(null)
const eps = ref([])
const epForm = ref({})

async function load() {
  const r = await admin.dramas(page.value, keyword.value)
  list.value = r.list || []
  total.value = r.total
}

onMounted(async () => {
  cats.value = await admin.categories()
  await load()
})

function openCreate() {
  form.value = { title: '', description: '', category_id: null, cover: '', banner: '', tags: '', is_featured: 0, is_completed: 0, is_hot: 0, status: 1, sort: 0 }
  dialog.value = true
}
function openEdit(row) {
  form.value = { ...row }
  dialog.value = true
}
async function save() {
  if (!form.value.title) return
  saving.value = true
  try {
    if (form.value.id) await admin.updateDrama(form.value.id, form.value)
    else await admin.createDrama(form.value)
    dialog.value = false
    ElMessage.success(t('saved'))
    await load()
  } catch (e) { ElMessage.error(e.message) } finally { saving.value = false }
}
async function remove(row) {
  await ElMessageBox.confirm(t('deleteDramaConfirm', { title: row.title }), t('warning'), {
    type: 'warning',
    confirmButtonText: t('del'),
    cancelButtonText: t('cancel'),
  })
  await admin.deleteDrama(row.id)
  ElMessage.success(t('deleted'))
  load()
}

async function openEpisodes(row) {
  epDrama.value = row
  eps.value = await admin.episodes(row.id)
  epDialog.value = true
}
function openEpCreate() {
  epForm.value = { drama_id: epDrama.value.id, ep_index: eps.value.length + 1, title: '', video_url: '', duration_sec: 0, price_coins: 0, status: 1 }
}
function openEpEdit(e) {
  epForm.value = { ...e }
}
async function saveEp() {
  try {
    await admin.saveEpisode(epForm.value)
    ElMessage.success(t('saved'))
    eps.value = await admin.episodes(epDrama.value.id)
    epForm.value = {}
  } catch (e) { ElMessage.error(e.message) }
}
async function removeEp(e) {
  await ElMessageBox.confirm(t('deleteEpisodeConfirm', { n: e.ep_index }), t('warning'), {
    type: 'warning',
    confirmButtonText: t('del'),
    cancelButtonText: t('cancel'),
  })
  await admin.deleteEpisode(e.id)
  eps.value = await admin.episodes(epDrama.value.id)
}
</script>

<template>
  <el-card>
    <div style="display:flex;gap:10px;margin-bottom:12px">
      <el-input v-model="keyword" :placeholder="t('searchTitle')" style="width:260px" clearable @keyup.enter="page = 1; load()" />
      <el-button @click="page = 1; load()">{{ t('search') }}</el-button>
      <el-button type="primary" @click="openCreate">{{ t('newDrama') }}</el-button>
    </div>

    <el-table :data="list" size="small">
      <el-table-column :label="t('cover')" width="70">
        <template #default="{ row }"><img :src="row.cover" style="width:44px;border-radius:4px" /></template>
      </el-table-column>
      <el-table-column prop="title" :label="t('title')" min-width="180" />
      <el-table-column prop="category_name" :label="t('colCategory')" width="100" />
      <el-table-column prop="episode_count" :label="t('colEpisodes')" width="80" />
      <el-table-column prop="views" :label="t('colViews')" width="90" />
      <el-table-column :label="t('colFlags')" width="160">
        <template #default="{ row }">
          <el-tag v-if="row.is_featured" size="small">{{ t('flagFeatured') }}</el-tag>
          <el-tag v-if="row.is_hot" size="small" type="danger">{{ t('flagHot') }}</el-tag>
          <el-tag v-if="row.is_completed" size="small" type="success">{{ t('flagDone') }}</el-tag>
          <el-tag v-if="row.status === 0" size="small" type="info">{{ t('flagHidden') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('actions')" width="240">
        <template #default="{ row }">
          <el-button size="small" @click="openEpisodes(row)">{{ t('colEpisodes') }}</el-button>
          <el-button size="small" @click="openEdit(row)">{{ t('edit') }}</el-button>
          <el-button size="small" type="danger" @click="remove(row)">{{ t('del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination style="margin-top:12px" layout="prev, pager, next" :total="total" :page-size="20" v-model:current-page="page" @current-change="load" />
  </el-card>

  <!-- drama dialog -->
  <el-dialog v-model="dialog" :title="form.id ? t('editDrama') : t('newDrama').replace('+ ', '')" width="560px">
    <el-form label-width="110px">
      <el-form-item :label="t('title')"><el-input v-model="form.title" /></el-form-item>
      <el-form-item :label="t('formCategory')">
        <el-select v-model="form.category_id" clearable style="width:100%">
          <el-option v-for="c in cats" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('formDescription')"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
      <el-form-item :label="t('formCoverUrl')"><el-input v-model="form.cover" placeholder="/static/posters/p1.svg" /></el-form-item>
      <el-form-item :label="t('formBannerUrl')"><el-input v-model="form.banner" /></el-form-item>
      <el-form-item :label="t('formTags')"><el-input v-model="form.tags" :placeholder="t('formTagsPh')" /></el-form-item>
      <el-form-item :label="t('formFlags')">
        <el-checkbox v-model="form.is_featured" :true-value="1" :false-value="0">{{ t('flagFeatured') }}</el-checkbox>
        <el-checkbox v-model="form.is_hot" :true-value="1" :false-value="0">{{ t('flagHot') }}</el-checkbox>
        <el-checkbox v-model="form.is_completed" :true-value="1" :false-value="0">{{ t('flagDone') }}</el-checkbox>
      </el-form-item>
      <el-form-item :label="t('status')">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" :active-text="t('published')" :inactive-text="t('hidden')" />
      </el-form-item>
      <el-form-item :label="t('formSort')"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialog = false">{{ t('cancel') }}</el-button>
      <el-button type="primary" :loading="saving" @click="save">{{ t('save') }}</el-button>
    </template>
  </el-dialog>

  <!-- episodes dialog -->
  <el-dialog v-model="epDialog" :title="t('episodesOf', { title: epDrama?.title || '' })" width="720px">
    <el-button size="small" type="primary" @click="openEpCreate">{{ t('addEpisode') }}</el-button>
    <el-table :data="eps" size="small" style="margin-top:10px">
      <el-table-column prop="ep_index" label="#" width="50" />
      <el-table-column prop="title" :label="t('epTitle')" min-width="160" />
      <el-table-column :label="t('epPrice')" width="90">
        <template #default="{ row }">{{ row.price_coins === 0 ? t('free') : row.price_coins }}</template>
      </el-table-column>
      <el-table-column :label="t('actions')" width="140">
        <template #default="{ row }">
          <el-button size="small" @click="openEpEdit(row)">{{ t('edit') }}</el-button>
          <el-button size="small" type="danger" @click="removeEp(row)">{{ t('del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>

  <!-- episode form dialog -->
  <el-dialog :model-value="!!epForm.ep_index || epForm.ep_index === 0" :title="epForm.id ? t('editEpisode') : t('newEpisode')" width="480px" @close="epForm = {}">
    <el-form label-width="100px">
      <el-form-item :label="t('epIndex')"><el-input-number v-model="epForm.ep_index" :min="1" /></el-form-item>
      <el-form-item :label="t('epTitle')"><el-input v-model="epForm.title" /></el-form-item>
      <el-form-item :label="t('videoUrl')"><el-input v-model="epForm.video_url" placeholder="https://…mp4" /></el-form-item>
      <el-form-item :label="t('duration')"><el-input-number v-model="epForm.duration_sec" :min="0" /></el-form-item>
      <el-form-item :label="t('epPriceLabel')"><el-input-number v-model="epForm.price_coins" :min="0" /> {{ t('freeHint') }}</el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="epForm = {}">{{ t('cancel') }}</el-button>
      <el-button type="primary" @click="saveEp">{{ t('save') }}</el-button>
    </template>
  </el-dialog>
</template>
