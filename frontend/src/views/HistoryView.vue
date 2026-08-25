<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'

const list = ref(null)
onMounted(async () => {
  list.value = await api.history()
})

function fmtDur(s) {
  return Math.floor(s / 60) + ':' + String(s % 60).padStart(2, '0')
}
</script>

<template>
  <div class="page">
    <button class="back" @click="$router.back()">‹</button>
    <div class="section-head"><h3>🕘 Watch History</h3></div>
    <div v-if="list && list.length">
      <router-link v-for="h in list" :key="h.episode_id" :to="`/player/${h.drama_id}/${h.episode_id}`" class="hrow">
        <img :src="h.cover" alt="" />
        <div class="hi">
          <div class="ht">{{ h.title }}</div>
          <div class="hm">Ep {{ h.ep_index }} · watched to {{ fmtDur(h.position_sec) }}</div>
          <div class="bar"><i :style="{ width: Math.min(100, (h.position_sec / (h.duration_sec || 1)) * 100) + '%' }"></i></div>
        </div>
        <span class="go">▶</span>
      </router-link>
    </div>
    <div v-else-if="list" class="empty">Nothing watched yet</div>
    <div v-else class="empty">Loading…</div>
  </div>
</template>

<style scoped>
.back { font-size: 26px; color: var(--muted); }
.hrow { display: flex; gap: 12px; align-items: center; padding: 10px 0; border-bottom: 1px solid var(--line); }
.hrow img { width: 64px; border-radius: 8px; aspect-ratio: 3/4; object-fit: cover; }
.hi { flex: 1; min-width: 0; }
.ht { font-weight: 700; font-size: 13.5px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.hm { color: var(--muted); font-size: 11px; margin: 4px 0; }
.bar { height: 3px; background: var(--bg-elev2); border-radius: 2px; overflow: hidden; }
.bar i { display: block; height: 100%; background: var(--accent); }
.go { color: var(--muted); font-size: 18px; }
</style>
