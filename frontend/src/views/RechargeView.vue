<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { store, refreshMe, fmtCoins } from '../store'

const route = useRoute()
const data = ref(null)
const tab = ref(route.query.tab === 'vip' ? 'vip' : 'coins')
const busy = ref(false)

onMounted(async () => {
  data.value = await api.store()
})

function cents(c) { return '$' + (c / 100).toFixed(2) }

// Dev flow: create order then settle it through the mock-pay endpoint.
// In production replace mockPay with a redirect to Stripe/PayPal checkout.
async function buy(kind, pkg) {
  if (busy.value) return
  busy.value = true
  try {
    const order = await api.createOrder(kind, pkg.id)
    await api.mockPay(order.order_no)
    await refreshMe()
    window.$toast(`Payment complete! Balance ${fmtCoins(store.user.coins)}`)
  } catch (e) {
    window.$toast(e.message)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="page" v-if="data">
    <button class="back" @click="$router.back()">‹</button>
    <div class="section-head"><h3>Store</h3></div>

    <div class="tabs">
      <button :class="{ on: tab === 'coins' }" @click="tab = 'coins'">🪙 Coins</button>
      <button :class="{ on: tab === 'vip' }" @click="tab = 'vip'">👑 VIP</button>
    </div>

    <div v-if="tab === 'coins'">
      <div class="bal">Current balance: <b class="coin">{{ fmtCoins(store.user?.coins ?? 0) }}</b></div>
      <div class="pkgs">
        <div v-for="p in data.coin_packages" :key="p.id" class="pkg" :class="{ best: p.tag }">
          <div v-if="p.tag" class="tag">{{ p.tag }}</div>
          <div class="coins">{{ fmtCoins(p.coins + p.bonus_coins) }}</div>
          <div class="sub">
            <span>{{ fmtCoins(p.coins) }} coins</span>
            <span v-if="p.bonus_coins" class="bonus">+{{ fmtCoins(p.bonus_coins) }} bonus</span>
          </div>
          <button class="btn btn-gold" :disabled="busy" @click="buy('coins', p)">{{ cents(p.price_cents) }}</button>
        </div>
      </div>
    </div>

    <div v-else>
      <div class="vip-hero card">
        <div class="vh-t">👑 Viewly VIP</div>
        <ul class="vh-list">
          <li>✓ Watch every episode free — no coins needed</li>
          <li>✓ Ad-free viewing experience</li>
          <li>✓ Priority access to new releases</li>
        </ul>
      </div>
      <div class="pkgs">
        <div v-for="p in data.vip_plans" :key="p.id" class="pkg" :class="{ best: p.tag }">
          <div v-if="p.tag" class="tag">{{ p.tag }}</div>
          <div class="coins small">{{ p.days }} days</div>
          <div class="sub"><span>{{ p.label }}</span></div>
          <button class="btn btn-gold" :disabled="busy" @click="buy('vip', p)">{{ cents(p.price_cents) }}</button>
        </div>
      </div>
    </div>

    <div class="terms">
      <p><b>Terms</b></p>
      <p>1. Coins are used to unlock premium episodes and never expire.</p>
      <p>2. VIP grants unlimited access during the subscription period.</p>
      <p>3. Dev environment uses mock payment; production integrates Stripe/PayPal.</p>
    </div>
  </div>
  <div v-else class="empty" style="padding-top:120px">Loading…</div>
</template>

<style scoped>
.back { font-size: 26px; color: var(--muted); }
.tabs { display: flex; gap: 8px; margin-bottom: 16px; }
.tabs button {
  flex: 1;
  padding: 10px;
  border-radius: 10px;
  background: var(--bg-elev);
  font-weight: 700;
  color: var(--muted);
}
.tabs button.on { background: linear-gradient(135deg, var(--accent), var(--accent-2)); color: #fff; }
.bal { color: var(--muted); font-size: 13px; margin-bottom: 12px; }
.pkgs { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.pkg {
  position: relative;
  background: var(--bg-elev);
  border: 1.5px solid var(--line);
  border-radius: 14px;
  padding: 18px 14px 14px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  overflow: hidden;
}
.pkg.best { border-color: var(--gold); }
.pkg .tag {
  position: absolute;
  top: 8px; right: -1px;
  background: linear-gradient(135deg, #f6b64c, #e08b1f);
  color: #3a2400;
  font-size: 9px;
  font-weight: 900;
  padding: 3px 9px;
  border-radius: 8px 0 0 8px;
}
.pkg .coins { font-size: 25px; font-weight: 900; color: var(--gold); }
.pkg .coins.small { color: var(--text); font-size: 21px; }
.pkg .sub { display: flex; flex-direction: column; align-items: center; color: var(--muted); font-size: 11px; }
.pkg .sub .bonus { color: var(--green); font-weight: 700; }
.pkg .btn { width: 100%; margin-top: 6px; }
.vip-hero { margin-bottom: 12px; }
.vh-t { font-weight: 900; color: var(--gold); font-size: 16px; margin-bottom: 8px; }
.vh-list { margin: 0; padding-left: 18px; color: var(--muted); font-size: 12.5px; line-height: 2; }
.terms { color: var(--muted); font-size: 11.5px; line-height: 1.9; margin-top: 20px; }
.terms p { margin: 0; }
</style>
