<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { admin, tenantSlug, getRole } from '../api'
import { t } from '../i18n'

const list = ref([])
const isSuper = getRole() === 'super'
const currentSlug = tenantSlug()
const form = ref({ name: '', slug: '', admin_username: '', admin_password: '' })
const busy = ref(false)

async function load() {
  list.value = await admin.tenants()
}
onMounted(load)

// super admins manage tenant sites by switching the tenant header
function switchTo(slug) {
  localStorage.setItem('viewly_tenant', slug)
  location.reload()
}

async function create() {
  if (!form.value.name || !form.value.slug || !form.value.admin_username || !form.value.admin_password) return
  busy.value = true
  try {
    await admin.createTenant(form.value)
    ElMessage.success(t('saved'))
    form.value = { name: '', slug: '', admin_username: '', admin_password: '' }
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <el-card v-if="isSuper">
    <div class="form-row">
      <el-input v-model="form.name" placeholder="Tenant name" style="width:160px" />
      <el-input v-model="form.slug" placeholder="slug (subdomain)" style="width:160px" />
      <el-input v-model="form.admin_username" placeholder="admin username" style="width:140px" />
      <el-input v-model="form.admin_password" type="password" placeholder="admin password" style="width:140px" show-password />
      <el-button type="primary" :loading="busy" @click="create">Create Tenant</el-button>
    </div>
    <el-table :data="list" size="small">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="Name" min-width="140" />
      <el-table-column label="Site">
        <template #default="{ row }">
          <el-link :href="`http://${row.slug}.localhost:5173`" target="_blank">{{ row.slug }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="dramas" label="Dramas" width="80" />
      <el-table-column prop="users" label="Users" width="80" />
      <el-table-column label="Status" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? 'Active' : 'Disabled' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Actions" width="120">
        <template #default="{ row }">
          <el-button size="small" :disabled="row.slug === currentSlug" @click="switchTo(row.slug)">
            {{ row.slug === currentSlug ? 'Current' : 'Manage' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
  <el-card v-else>
    <p style="color:#999;margin:0">Platform super admin only.</p>
  </el-card>
</template>

<style scoped>
.form-row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
</style>
