<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { admin } from '../api'
import { t } from '../i18n'

const list = ref([])
const form = ref({ image: '', drama_id: 0, sort: 0 })

async function load() {
  list.value = await admin.banners()
}
onMounted(load)

async function save() {
  if (!form.value.image) return
  await admin.saveBanner(form.value)
  form.value = { image: '', drama_id: 0, sort: 0 }
  ElMessage.success(t('saved'))
  load()
}
async function remove(b) {
  await ElMessageBox.confirm(t('deleteBannerConfirm'), t('warning'), {
    type: 'warning',
    confirmButtonText: t('del'),
    cancelButtonText: t('cancel'),
  })
  await admin.deleteBanner(b.id)
  load()
}
</script>

<template>
  <el-card>
    <div style="display:flex;gap:10px;margin-bottom:12px">
      <el-input v-model="form.image" :placeholder="t('imageUrl')" style="width:300px" />
      <el-input-number v-model="form.drama_id" :min="0" :placeholder="t('dramaId')" />
      <el-input-number v-model="form.sort" :min="0" />
      <el-button type="primary" @click="save">{{ form.id ? t('update') : t('add') }}</el-button>
    </div>
    <el-table :data="list" size="small">
      <el-table-column :label="t('imageUrl')" width="120">
        <template #default="{ row }"><img :src="row.image" style="width:96px;border-radius:4px" /></template>
      </el-table-column>
      <el-table-column prop="drama_id" :label="t('dramaId')" width="100" />
      <el-table-column prop="sort" :label="t('sort')" width="80" />
      <el-table-column :label="t('actions')" width="160">
        <template #default="{ row }">
          <el-button size="small" @click="form = { ...row }">{{ t('edit') }}</el-button>
          <el-button size="small" type="danger" @click="remove(row)">{{ t('del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>
