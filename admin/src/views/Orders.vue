<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { admin } from '../api'
import { t } from '../i18n'

const list = ref([])
const total = ref(0)
const page = ref(1)
const status = ref('')

async function load() {
  const r = await admin.orders(page.value, status.value)
  list.value = r.list || []
  total.value = r.total
}
onMounted(load)

async function markPaid(row) {
  await admin.markPaid(row.order_no)
  ElMessage.success(t('settled'))
  load()
}
</script>

<template>
  <el-card>
    <div style="display:flex;gap:10px;margin-bottom:12px">
      <el-select v-model="status" clearable :placeholder="t('allStatus')" style="width:150px" @change="page = 1; load()">
        <el-option :label="t('pending')" value="pending" />
        <el-option :label="t('paid')" value="paid" />
      </el-select>
    </div>
    <el-table :data="list" size="small">
      <el-table-column prop="order_no" :label="t('colOrderNo')" min-width="190" />
      <el-table-column prop="user_id" :label="t('colUser')" width="70" />
      <el-table-column prop="kind" :label="t('colKind')" width="70" />
      <el-table-column :label="t('colContent')" min-width="140">
        <template #default="{ row }">
          <span v-if="row.kind === 'coins'">{{ row.coins + row.bonus_coins }} 🪙</span>
          <span v-else>{{ t('vipDays', { n: row.days }) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('colAmount')" width="90">
        <template #default="{ row }">${{ (row.amount_cents / 100).toFixed(2) }}</template>
      </el-table-column>
      <el-table-column :label="t('status')" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'paid' ? 'success' : 'warning'" size="small">
            {{ row.status === 'paid' ? t('paid') : t('pending') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" :label="t('colCreated')" width="170" />
      <el-table-column :label="t('actions')" width="120">
        <template #default="{ row }">
          <el-button v-if="row.status === 'pending'" size="small" type="primary" @click="markPaid(row)">{{ t('markPaid') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination style="margin-top:12px" layout="prev, pager, next" :total="total" :page-size="20" v-model:current-page="page" @current-change="load" />
  </el-card>
</template>
