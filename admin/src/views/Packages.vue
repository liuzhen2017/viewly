<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { admin } from '../api'
import { t } from '../i18n'

const packs = ref([])
const plans = ref([])
const coinForm = ref({})
const vipForm = ref({})

async function load() {
  const r = await admin.packages()
  packs.value = r.coin_packages || []
  plans.value = r.vip_plans || []
}
onMounted(load)

async function saveCoins() {
  if (!coinForm.value.coins || !coinForm.value.price_cents) return
  await admin.savePackage('coins', coinForm.value)
  coinForm.value = {}
  ElMessage.success(t('saved'))
  load()
}
async function saveVip() {
  if (!vipForm.value.days || !vipForm.value.price_cents) return
  await admin.savePackage('vip', vipForm.value)
  vipForm.value = {}
  ElMessage.success(t('saved'))
  load()
}
async function remove(kind, row) {
  await ElMessageBox.confirm(t('deleteItemConfirm'), t('warning'), {
    type: 'warning',
    confirmButtonText: t('del'),
    cancelButtonText: t('cancel'),
  })
  await admin.deletePackage(kind, row.id)
  load()
}
</script>

<template>
  <el-row :gutter="14">
    <el-col :span="12">
      <el-card>
        <template #header>{{ t('coinPackages') }}</template>
        <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap">
          <el-input-number v-model="coinForm.coins" :min="1" :placeholder="t('coins')" />
          <el-input-number v-model="coinForm.bonus_coins" :min="0" :placeholder="t('bonus')" />
          <el-input-number v-model="coinForm.price_cents" :min="1" :placeholder="t('priceCents')" />
          <el-input v-model="coinForm.label" :placeholder="t('label')" style="width:130px" />
          <el-input v-model="coinForm.tag" :placeholder="t('tag')" style="width:110px" />
          <el-button type="primary" @click="saveCoins">{{ coinForm.id ? t('update') : t('add') }}</el-button>
        </div>
        <el-table :data="packs" size="small">
          <el-table-column prop="id" label="ID" width="50" />
          <el-table-column :label="t('coins')">
            <template #default="{ row }">{{ row.coins }}<span v-if="row.bonus_coins" style="color:#67c23a"> +{{ row.bonus_coins }}</span></template>
          </el-table-column>
          <el-table-column :label="t('colAmount')">
            <template #default="{ row }">${{ (row.price_cents / 100).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column prop="tag" :label="t('tag')" width="100" />
          <el-table-column :label="t('actions')" width="140">
            <template #default="{ row }">
              <el-button size="small" @click="coinForm = { ...row }">{{ t('edit') }}</el-button>
              <el-button size="small" type="danger" @click="remove('coins', row)">{{ t('del') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </el-col>
    <el-col :span="12">
      <el-card>
        <template #header>{{ t('vipPlans') }}</template>
        <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap">
          <el-input-number v-model="vipForm.days" :min="1" :placeholder="t('days')" />
          <el-input-number v-model="vipForm.price_cents" :min="1" :placeholder="t('priceCents')" />
          <el-input v-model="vipForm.label" :placeholder="t('label')" style="width:130px" />
          <el-input v-model="vipForm.tag" :placeholder="t('tag')" style="width:110px" />
          <el-button type="primary" @click="saveVip">{{ vipForm.id ? t('update') : t('add') }}</el-button>
        </div>
        <el-table :data="plans" size="small">
          <el-table-column prop="id" label="ID" width="50" />
          <el-table-column prop="days" :label="t('days')" width="70" />
          <el-table-column :label="t('colAmount')">
            <template #default="{ row }">${{ (row.price_cents / 100).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column prop="tag" :label="t('tag')" width="100" />
          <el-table-column :label="t('actions')" width="140">
            <template #default="{ row }">
              <el-button size="small" @click="vipForm = { ...row }">{{ t('edit') }}</el-button>
              <el-button size="small" type="danger" @click="remove('vip', row)">{{ t('del') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </el-col>
  </el-row>
</template>
