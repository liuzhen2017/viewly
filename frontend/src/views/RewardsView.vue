<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { store, refreshMe, fmtCoins } from '../store'

const data = ref(null)
const busy = ref(false)

onMounted(load)

async function load() {
  data.value = await api.rewards()
}

async function doCheckin() {
  if (busy.value) return
  busy.value = true
  try {
    const r = await api.checkin()
    window.$toast(`+${r.coins} coins! Balance ${r.balance}`)
    await Promise.all([load(), refreshMe()])
  } catch (e) {
    window.$toast(e.message)
  } finally {
    busy.value = false
  }
}

async function claim(t) {
  if (!t.claimable || busy.value) return
  busy.value = true
  try {
    const r = await api.claimTask(t.key)
    window.$toast(`+${r.coins} coins!`)
    await Promise.all([load(), refreshMe()])
  } catch (e) {
    window.$toast(e.message)
  } finally {
    busy.value = false
  }
}

function toastGo() {
  window.$toast('Watch / share / like dramas to complete the task')
}
</script>

<template>
  <div class="page" v-if="data">
    <div class="head-card">
      <div class="balance">
        <span class="label">My Coins</span>
        <span class="num">{{ fmtCoins(store.user?.coins ?? data.coins) }}</span>
      </div>
      <router-link to="/recharge" class="btn btn-gold btn-sm">Top Up</router-link>
    </div>

    <!-- 7-day check-in -->
    <div class="card checkin-card">
      <div class="section-head" style="margin-top:0">
        <h3>📅 Welfare Check-in</h3>
        <span class="badge badge-new">Day {{ data.checkin.cycle_day }}/7</span>
      </div>
      <div class="days">
        <div v-for="d in data.checkin.days" :key="d.day" class="day" :class="{ done: d.done, current: d.current && !data.checkin.today_done }">
          <span class="dc">+{{ d.coins }}</span>
          <span class="dl">{{ d.day === 7 ? 'Day 7' : 'Day ' + d.day }}</span>
        </div>
      </div>
      <button
        class="btn btn-primary claim-btn" :disabled="data.checkin.today_done || busy" @click="doCheckin"
      >{{ data.checkin.today_done ? '✓ Checked in today' : 'Check In Today' }}</button>
    </div>

    <!-- daily tasks -->
    <div class="section-head"><h3>🎯 Daily Benefits</h3></div>
    <div class="card task" v-for="t in data.tasks.filter(x => x.group === 'daily')" :key="t.key">
      <div class="ti">
        <div class="tt">{{ t.title }}</div>
        <div class="tm" v-if="t.threshold > 1">{{ t.progress }}/{{ t.threshold }}</div>
      </div>
      <div class="tr">
        <span class="coin">+{{ t.coins }}</span>
        <button v-if="t.rewarded" class="btn btn-sm" disabled>Claimed</button>
        <button v-else-if="t.claimable" class="btn btn-gold btn-sm" @click="claim(t)">Claim</button>
        <button v-else class="btn btn-ghost btn-sm" @click="toastGo">
          {{ t.key === 'share' ? 'Share' : t.key === 'like' ? 'Like' : t.key === 'favorite' ? 'Favorite' : 'Go' }}
        </button>
      </div>
    </div>

    <!-- social tasks -->
    <div class="section-head"><h3>🎊 New Player Benefits</h3></div>
    <div class="card task" v-for="t in data.tasks.filter(x => x.group === 'social')" :key="t.key">
      <div class="ti">
        <div class="tt">{{ t.title }}</div>
        <div class="tm">One-time reward</div>
      </div>
      <div class="tr">
        <span class="coin">+{{ t.coins }}</span>
        <button v-if="t.rewarded" class="btn btn-sm" disabled>Claimed</button>
        <button v-else class="btn btn-gold btn-sm" @click="claim(t)">Claim</button>
      </div>
    </div>

    <div class="rules">
      <p><b>Welfare Description</b></p>
      <p>1. Sign in daily to claim benefits (7-day cycle).</p>
      <p>2. Complete tasks to claim coin rewards. Tasks reset daily.</p>
      <p>3. Coins can unlock premium episodes.</p>
    </div>
  </div>
  <div v-else class="empty" style="padding-top:120px">Loading…</div>
</template>

<style scoped>
.head-card {
  display: flex; align-items: center; justify-content: space-between;
  background: linear-gradient(135deg, #241a08, #17171f);
  border: 1px solid #3a2c10;
  border-radius: var(--radius);
  padding: 16px;
}
.balance { display: flex; flex-direction: column; }
.balance .label { color: var(--muted); font-size: 12px; }
.balance .num { font-size: 26px; font-weight: 900; color: var(--gold); }
.checkin-card { margin-top: 12px; }
.days { display: grid; grid-template-columns: repeat(7, 1fr); gap: 6px; margin: 4px 0 14px; }
.day {
  display: flex; flex-direction: column; align-items: center; gap: 3px;
  background: var(--bg-elev2);
  border-radius: 10px;
  padding: 9px 2px;
  font-size: 10px;
}
.day .dc { font-weight: 800; color: var(--gold); }
.day .dl { color: var(--muted); }
.day.done { background: rgba(53,192,123,.12); }
.day.done .dc { color: var(--green); }
.day.current { outline: 2px solid var(--accent); }
.claim-btn { width: 100%; padding: 12px; font-size: 15px; }
.task { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.ti { min-width: 0; }
.tt { font-weight: 700; font-size: 14px; }
.tm { color: var(--muted); font-size: 11.5px; margin-top: 2px; }
.tr { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.rules { color: var(--muted); font-size: 12px; line-height: 1.9; margin-top: 18px; }
.rules p { margin: 0; }
</style>
