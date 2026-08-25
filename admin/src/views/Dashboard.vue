<script setup>
import { ref, onMounted, computed } from 'vue'
import { admin } from '../api'
import { t } from '../i18n'

const s = ref(null)
onMounted(async () => {
  s.value = await admin.stats()
})
const money = (c) => '$' + (c / 100).toFixed(2)

const cards = computed(() => s.value ? [
  { label: t('kpiUsers'), value: s.value.users, sub: t('subNewUsers', { n: s.value.new_users_today }) },
  { label: t('kpiPaidOrders'), value: s.value.paid_orders, sub: t('subOrdersTotal', { n: s.value.orders }) },
  { label: t('kpiRevenue'), value: money(s.value.revenue_cents), sub: t('subAllTime') },
  { label: t('kpiUnlocks'), value: s.value.unlocks_today, sub: t('subCheckins', { n: s.value.checkins_today }) },
] : [])
</script>

<template>
  <div v-if="s">
    <el-row :gutter="14">
      <el-col :span="6" v-for="card in cards" :key="card.label">
        <el-card>
          <div class="kpi-label">{{ card.label }}</div>
          <div class="kpi-value">{{ card.value }}</div>
          <div class="kpi-sub">{{ card.sub }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card style="margin-top:14px">
      <template #header>{{ t('last7days') }}</template>
      <el-table :data="s.revenue_series" size="small">
        <el-table-column :label="t('colDay')">
          <template #default="{ row }">{{ new Date(row.day).toLocaleDateString() }}</template>
        </el-table-column>
        <el-table-column prop="orders" :label="t('colOrders')" width="120" />
        <el-table-column :label="t('colRevenue')">
          <template #default="{ row }">{{ money(row.cents) }}</template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card style="margin-top:14px">
      <template #header>{{ t('contentCard') }}</template>
      <p>{{ t('contentSummary', { d: s.dramas, e: s.episodes }) }}</p>
    </el-card>
  </div>
</template>

<style scoped>
.kpi-label { color: #888; font-size: 13px; }
.kpi-value { font-size: 28px; font-weight: 800; margin: 6px 0 2px; }
.kpi-sub { color: #999; font-size: 12px; }
</style>
