<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { admin } from '../api'
import { t } from '../i18n'

const list = ref([])
const form = ref({ name: '', sort: 0 })

async function load() {
  list.value = await admin.categories()
}
onMounted(load)

async function save() {
  if (!form.value.name) return
  await admin.saveCategory(form.value)
  form.value = { name: '', sort: 0 }
  ElMessage.success(t('saved'))
  load()
}
async function remove(c) {
  await ElMessageBox.confirm(t('deleteCategoryConfirm', { name: c.name }), t('warning'), {
    type: 'warning',
    confirmButtonText: t('del'),
    cancelButtonText: t('cancel'),
  })
  await admin.deleteCategory(c.id)
  load()
}
</script>

<template>
  <el-card>
    <div style="display:flex;gap:10px;margin-bottom:12px">
      <el-input v-model="form.name" :placeholder="t('name')" style="width:200px" />
      <el-input-number v-model="form.sort" :min="0" />
      <el-button type="primary" @click="save">{{ form.id ? t('update') : t('add') }}</el-button>
    </div>
    <el-table :data="list" size="small">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" :label="t('name')" />
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
