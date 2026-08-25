<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { admin } from '../api'
import { t } from '../i18n'

const list = ref([])
const total = ref(0)
const page = ref(1)
const keyword = ref('')
const adjustVisible = ref(false)
const adjustForm = ref({ user_id: 0, coins: 100, remark: '' })

async function load() {
  const r = await admin.users(page.value, keyword.value)
  list.value = r.list || []
  total.value = r.total
}
onMounted(load)

function openAdjust(row) {
  adjustForm.value = { user_id: row.id, coins: 100, remark: '' }
  adjustVisible.value = true
}
async function doAdjust() {
  try {
    const r = await admin.adjust(adjustForm.value.user_id, adjustForm.value.coins, adjustForm.value.remark)
    ElMessage.success(t('adjustedOk', { n: r.balance }))
    adjustVisible.value = false
    load()
  } catch (e) { ElMessage.error(e.message) }
}
</script>

<template>
  <el-card>
    <div style="display:flex;gap:10px;margin-bottom:12px">
      <el-input v-model="keyword" :placeholder="t('searchUser')" style="width:260px" clearable @keyup.enter="page = 1; load()" />
      <el-button @click="page = 1; load()">{{ t('search') }}</el-button>
    </div>
    <el-table :data="list" size="small">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="nickname" :label="t('colNickname')" min-width="140" />
      <el-table-column prop="email" :label="t('colEmail')" min-width="160" />
      <el-table-column :label="t('colGuest')" width="70">
        <template #default="{ row }">{{ row.is_guest ? '✓' : '' }}</template>
      </el-table-column>
      <el-table-column prop="coins" :label="t('colCoins')" width="100" />
      <el-table-column label="VIP" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.is_vip" size="small" type="warning">VIP</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" :label="t('colJoined')" width="170" />
      <el-table-column :label="t('actions')" width="110">
        <template #default="{ row }">
          <el-button size="small" @click="openAdjust(row)">{{ t('adjust') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination style="margin-top:12px" layout="prev, pager, next" :total="total" :page-size="20" v-model:current-page="page" @current-change="load" />
  </el-card>

  <el-dialog v-model="adjustVisible" :title="t('adjustCoins')" width="420px">
    <el-form label-width="90px">
      <el-form-item :label="t('userID')">{{ adjustForm.user_id }}</el-form-item>
      <el-form-item :label="t('coinsDelta')">
        <el-input-number v-model="adjustForm.coins" />
      </el-form-item>
      <el-form-item :label="t('remark')"><el-input v-model="adjustForm.remark" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="adjustVisible = false">{{ t('cancel') }}</el-button>
      <el-button type="primary" @click="doAdjust">{{ t('apply') }}</el-button>
    </template>
  </el-dialog>
</template>
