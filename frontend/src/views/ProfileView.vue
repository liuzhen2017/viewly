<script setup>
import { store, fmtCoins } from '../store'
</script>

<template>
  <div class="page">
    <!-- account card -->
    <div class="user-card">
      <div class="avatar">
        <img v-if="store.user?.avatar" :src="store.user.avatar" alt="" />
        <span v-else class="def">{{ (store.user?.nickname || '?').slice(0, 1).toUpperCase() }}</span>
      </div>
      <div class="uinfo">
        <div class="nick">
          {{ store.user?.nickname || 'Guest' }}
          <span v-if="store.user?.is_vip" class="vip">VIP</span>
        </div>
        <div class="uid">{{ store.user?.email || 'User ID: ' + (store.user?.id || '—') }}</div>
      </div>
      <router-link v-if="!store.user?.email" to="/login" class="btn btn-primary btn-sm">Sign In</router-link>
    </div>

    <!-- vip banner -->
    <router-link to="/recharge?tab=vip" class="vip-banner">
      <div>
        <div class="vt">👑 VIP Subscription</div>
        <div class="vs">{{ store.user?.is_vip ? 'Active until ' + new Date(store.user.vip_expire_at).toLocaleDateString() : 'Watch all dramas free · Cancel anytime' }}</div>
      </div>
      <span class="go">GO ›</span>
    </router-link>

    <!-- wallet -->
    <div class="wallet card">
      <div class="wb">
        <div class="wl">🪙 Coins</div>
        <div class="wn">{{ fmtCoins(store.user?.coins ?? 0) }}</div>
      </div>
      <div class="wbtns">
        <router-link to="/recharge" class="btn btn-gold btn-sm">Top Up</router-link>
      </div>
    </div>

    <!-- menu -->
    <div class="menu card">
      <router-link class="mi" to="/wallet"><span>💰 Wallet & Transactions</span><i>›</i></router-link>
      <router-link class="mi" to="/recharge"><span>💎 Recharge / VIP</span><i>›</i></router-link>
      <router-link class="mi" to="/history"><span>🕘 Watch History</span><i>›</i></router-link>
      <router-link class="mi" to="/playlist"><span>📚 My Playlist</span><i>›</i></router-link>
      <router-link v-if="!store.user?.email" class="mi" to="/login"><span>📧 Bind Email (keep your coins safe)</span><i>›</i></router-link>
    </div>

    <div class="foot">
      <router-link to="/privacy">Privacy Policy</router-link> ·
      <router-link to="/terms">Terms of Service</router-link>
      <div>Viewly · Short Drama Platform</div>
    </div>
  </div>
</template>

<style scoped>
.user-card { display: flex; align-items: center; gap: 14px; padding: 6px 2px 18px; }
.avatar { width: 62px; height: 62px; border-radius: 50%; overflow: hidden; background: var(--bg-elev2); display: flex; align-items: center; justify-content: center; }
.avatar .def { font-size: 24px; font-weight: 800; color: var(--accent); }
.avatar img { width: 100%; height: 100%; object-fit: cover; }
.uinfo { flex: 1; }
.nick { font-size: 17px; font-weight: 800; display: flex; gap: 8px; align-items: center; }
.vip {
  font-size: 9px; padding: 2px 7px; border-radius: 5px;
  background: linear-gradient(135deg, #f6b64c, #e08b1f); color: #3a2400; font-weight: 900;
}
.uid { color: var(--muted); font-size: 11.5px; margin-top: 3px; }
.vip-banner {
  display: flex; justify-content: space-between; align-items: center;
  background: linear-gradient(120deg, #241605, #17171f 70%);
  border: 1px solid #3d2d0c;
  border-radius: var(--radius);
  padding: 14px 16px;
  margin-bottom: 10px;
}
.vt { font-weight: 800; color: var(--gold); font-size: 14.5px; }
.vs { color: var(--muted); font-size: 11.5px; margin-top: 3px; }
.go { color: var(--gold); font-weight: 800; }
.wallet { display: flex; justify-content: space-between; align-items: center; }
.wl { color: var(--muted); font-size: 12px; }
.wn { font-size: 24px; font-weight: 900; color: var(--gold); }
.menu { margin-top: 10px; padding: 4px 14px; }
.mi { display: flex; justify-content: space-between; align-items: center; padding: 14px 0; border-bottom: 1px solid var(--line); font-size: 14px; }
.mi:last-child { border-bottom: none; }
.mi i { color: var(--muted); font-style: normal; }
.foot { text-align: center; color: var(--muted); font-size: 11px; margin-top: 26px; opacity: .6; }
</style>
