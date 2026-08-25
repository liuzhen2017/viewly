<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { store, fmtCoins } from '../store'

const wallet = ref(null)
const list = ref([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)

onMounted(async () => {
  wallet.value = await api.wallet()
  await loadMore()
})

async function loadMore() {
  if (loading.value) return
  loading.value = true
  const r = await api.transactions(page.value)
  list.value.push(...r.list)
  total.value = r.total
  page.value++
  loading.value = false
}

const bizLabel = {
  signup: 'Signup bonus', checkin: 'Daily check-in', task: 'Task reward',
  recharge: 'Recharge', unlock: 'Unlock episode', admin: 'Admin adjustment',
}
</script>

<template>
  <div class="page">
    <button class="back" @click="$router.back()">‹</button>
    <div class="section-head"><h3>💰 Wallet</h3></div>

    <div v-if="wallet" class="card balance-card">
      <div class="row">
        <div><div class="l">Coins</div><div class="n">{{ fmtCoins(wallet.coins) }}</div></div>
        <div><div class="l">VIP</div><div class="n sm">{{ wallet.is_vip ? 'Active' : '—' }}</div></div>
      </div>
      <div class="stats">
        <span>Earned <b class="coin">+{{ fmtCoins(wallet.total_earned) }}</b></span>
        <span>Spent <b>{{ fmtCoins(wallet.total_spent) }}</b></span>
      </div>
      <router-link to="/recharge" class="btn btn-gold" style="width:100%;margin-top:12px">Top Up</router-link>
    </div>

    <div class="section-head"><h3>Transactions</h3></div>
    <div v-if="list.length" class="tx-list">
      <div v-for="t in list" :key="t.id" class="tx">
        <div class="ti">
          <div class="tt">{{ bizLabel[t.biz_type] || t.biz_type }}</div>
          <div class="tm">{{ t.remark || t.biz_id }} · {{ new Date(t.created_at).toLocaleString() }}</div>
        </div>
        <span class="amt" :class="{ plus: t.amount > 0 }">{{ t.amount > 0 ? '+' : '' }}{{ fmtCoins(t.amount) }}</span>
      </div>
      <button v-if="list.length < total" class="btn btn-ghost" style="width:100%;margin-top:10px" @click="loadMore">Load more</button>
    </div>
    <div v-else class="empty">No transactions yet</div>
  </div>
</template>

<style scoped>
.back { font-size: 26px; color: var(--muted); }
.balance-card .row { display: flex; gap: 28px; }
.balance-card .l { color: var(--muted); font-size: 12px; }
.balance-card .n { font-size: 26px; font-weight: 900; color: var(--gold); }
.balance-card .n.sm { font-size: 19px; color: var(--text); padding-top: 6px; display: inline-block; }
.stats { display: flex; gap: 20px; margin-top: 12px; color: var(--muted); font-size: 12px; }
.tx { display: flex; justify-content: space-between; align-items: center; padding: 11px 0; border-bottom: 1px solid var(--line); }
.tt { font-weight: 700; font-size: 13.5px; }
.tm { color: var(--muted); font-size: 11px; margin-top: 2px; }
.amt { font-weight: 800; }
.amt.plus { color: var(--green); }
</style>
