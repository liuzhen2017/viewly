<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { admin } from '../api'
import { t } from '../i18n'

const list = ref([])
const dramas = ref([])
const dialog = ref(false)
const busy = ref(false)
const imgInput = ref(null)
const uploading = ref(0)
const uploadPct = ref(0)
const form = ref({ image: '', drama_id: 0, sort: 0 })

async function load() {
  list.value = await admin.banners()
}
onMounted(async () => {
  load()
  try {
    const r = await admin.dramas(1, 100)
    dramas.value = r.list || []
  } catch { dramas.value = [] }
})

function openCreate() {
  form.value = { image: '', drama_id: 0, sort: (list.value.length + 1) * 10 }
  dialog.value = true
}
function openEdit(row) {
  form.value = { ...row }
  dialog.value = true
}
async function save() {
  if (!form.value.image) {
    ElMessage.warning(t('imageUrl'))
    return
  }
  busy.value = true
  try {
    await admin.saveBanner(form.value)
    dialog.value = false
    ElMessage.success(t('saved'))
    await load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    busy.value = false
  }
}
async function remove(b) {
  await ElMessageBox.confirm(t('deleteBannerConfirm'), t('warning'), {
    type: 'warning', confirmButtonText: t('del'), cancelButtonText: t('cancel'),
  })
  await admin.deleteBanner(b.id)
  ElMessage.success(t('deleted'))
  load()
}

async function uploadImage(file, onDone) {
  uploading.value++
  try {
    const r = await admin.uploadFile(file, pct => { uploadPct.value = pct })
    onDone(r.cdn_url)
    ElMessage.success(t('uploaded'))
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    uploading.value = 0
    uploadPct.value = 0
  }
}
function pickImage() { imgInput.value && imgInput.value.click() }
function onImagePicked(e) {
  const f = e.target.files && e.target.files[0]
  if (f) uploadImage(f, url => { form.value.image = url })
  e.target.value = ''
}
const dramaById = id => dramas.value.find(d => d.id === id)
</script>

<template>
  <el-card>
    <div style="display:flex;gap:10px;margin-bottom:12px">
      <el-button type="primary" @click="openCreate">+ {{ t('newBanner') }}</el-button>
    </div>
    <el-table :data="list" size="small">
      <el-table-column :label="t('preview')" width="180">
        <template #default="{ row }">
          <img :src="row.image" style="width:160px;height:75px;object-fit:cover;border-radius:6px" />
        </template>
      </el-table-column>
      <el-table-column :label="t('linkDrama')">
        <template #default="{ row }">
          {{ dramaById(row.drama_id)?.title || (row.drama_id ? '#' + row.drama_id : '—') }}
        </template>
      </el-table-column>
      <el-table-column prop="sort" :label="t('sort')" width="80" />
      <el-table-column :label="t('actions')" width="170">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">{{ t('edit') }}</el-button>
          <el-button size="small" type="danger" class="row-del" @click="remove(row)">{{ t('del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialog" :title="form.id ? t('editBanner') : t('newBanner')" width="520px">
    <el-form label-width="100px">
      <el-form-item :label="t('imageUrl')">
        <div style="display:flex;flex-direction:column;gap:8px;width:100%">
          <div style="display:flex;gap:8px;align-items:center">
            <el-button type="primary" :loading="uploading > 0" @click="pickImage">
              {{ uploading > 0 ? t('uploading') + ' ' + uploadPct + '%' : t('uploadImage') }}
            </el-button>
            <input ref="imgInput" type="file" hidden accept="image/*" @change="onImagePicked" />
            <span style="color:#999;font-size:12px">16:7</span>
          </div>
          <el-input v-model="form.image" placeholder="https://…" />
          <img v-if="form.image" :src="form.image" style="width:100%;max-height:180px;object-fit:cover;border-radius:8px" />
        </div>
      </el-form-item>
      <el-form-item :label="t('linkDrama')">
        <el-select v-model="form.drama_id" clearable style="width:100%">
          <el-option v-for="d in dramas" :key="d.id" :value="d.id" :label="d.title">
            <div style="display:flex;align-items:center;gap:8px">
              <img :src="d.cover" style="width:24px;height:32px;object-fit:cover;border-radius:3px" />
              <span>{{ d.title }}</span>
            </div>
          </el-option>
        </el-select>
      </el-form-item>
      <el-form-item :label="t('sort')">
        <el-input-number v-model="form.sort" :min="0" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialog = false">{{ t('cancel') }}</el-button>
      <el-button type="primary" :loading="busy" @click="save">{{ t('save') }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
:deep(.row-del) { margin-left: 14px; }
</style>
