<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { store, refreshMe, fmtCoins } from '../store'

const route = useRoute()
const router = useRouter()

const videoEl = ref(null)
const drama = ref(null)
const currentEp = ref(null)
const playing = ref(false)
const showUnlock = ref(false)
const unlockBusy = ref(false)
const cur = ref(0)
const dur = ref(0)
let progressTimer = null
let watchTimer = null

const currentIdx = computed(() =>
  drama.value ? drama.value.episodes.findIndex(e => e.id === Number(route.params.episodeId)) : -1
)
const prevEp = computed(() => currentIdx.value > 0 ? drama.value.episodes[currentIdx.value - 1] : null)
const nextEp = computed(() => {
  if (!drama.value) return null
  const eps = drama.value.episodes
  return currentIdx.value >= 0 && currentIdx.value < eps.length - 1 ? eps[currentIdx.value + 1] : null
})

onMounted(async () => {
  drama.value = await api.drama(route.params.dramaId)
  await tryPlay(route.params.episodeId)
})

onBeforeUnmount(() => {
  clearInterval(progressTimer)
  clearTimeout(watchTimer)
})

async function tryPlay(epId) {
  const ep = drama.value.episodes.find(e => e.id === Number(epId))
  if (!ep) return
  currentEp.value = ep
  try {
    const r = await api.play(ep.id)
    loadVideo(r.video_url)
  } catch (e) {
    if (e.code === 402 || e.message === 'locked') {
      showUnlock.value = true
    } else {
      window.$toast(e.message)
    }
  }
}

function loadVideo(url) {
  showUnlock.value = false
  const v = videoEl.value
  v.src = url
  v.play().catch(() => {})
  clearInterval(progressTimer)
  progressTimer = setInterval(report, 10000)
  // advance watch-count tasks shortly after playback starts
  watchTimer = setTimeout(report, 4000)
}

function report() {
  const v = videoEl.value
  if (!v || !v.src || !currentEp.value) return
  if (v.currentTime > 0) {
    api.progress(currentEp.value.id, Math.floor(v.currentTime), Math.floor(v.duration || 0)).catch(() => {})
  }
}

async function confirmUnlock() {
  if (!currentEp.value) return
  unlockBusy.value = true
  try {
    const r = await api.unlock(currentEp.value.id)
    await refreshMe()
    const ep = drama.value.episodes.find(e => e.id === currentEp.value.id)
    if (ep) { ep.accessible = true; ep.free = false }
    window.$toast(`Unlocked! ${r.coins} coins left`)
    loadVideo(r.video_url)
  } catch (e) {
    if (e.code === 2003) {
      router.push('/recharge')
    } else {
      window.$toast(e.message)
    }
  } finally {
    unlockBusy.value = false
  }
}

function switchEp(ep) {
  router.replace(`/player/${drama.value.id}/${ep.id}`)
  tryPlay(ep.id)
}

function onEnded() {
  report()
  if (nextEp.value) switchEp(nextEp.value)
}
function onTime() {
  const v = videoEl.value
  cur.value = v.currentTime || 0
  dur.value = v.duration || 0
}
function fmt(s) {
  s = Math.floor(s)
  return Math.floor(s / 60) + ':' + String(s % 60).padStart(2, '0')
}
</script>

<template>
  <div class="page player-page" v-if="drama">
    <div class="topbar">
      <button class="back" @click="$router.push(`/drama/${drama.id}`)">‹</button>
      <div class="title">
        <div class="t">{{ drama.title }}</div>
        <div class="ep-label" v-if="currentEp">Ep {{ currentEp.ep_index }} · {{ currentEp.title }}</div>
      </div>
      <router-link to="/wallet" class="coin">{{ fmtCoins(store.user?.coins ?? 0) }}</router-link>
    </div>

    <div class="video-wrap">
      <video
        ref="videoEl"
        playsinline
        controls
        @play="playing = true"
        @pause="report"
        @timeupdate="onTime"
        @ended="onEnded"
      ></video>
      <div v-if="!videoEl?.src" class="placeholder">
        <img v-if="currentEp" :src="drama.cover" alt="" />
        <div class="lock-overlay" v-if="showUnlock">
          <div class="lock-icon">🔒</div>
          <div class="lock-text">Ep {{ currentEp?.ep_index }} requires {{ currentEp?.price_coins }} coins</div>
          <div class="lock-actions">
            <button class="btn btn-primary" :disabled="unlockBusy" @click="confirmUnlock">
              Unlock · {{ currentEp?.price_coins }} 🪙
            </button>
            <router-link to="/recharge" class="btn btn-gold">Top Up</router-link>
          </div>
        </div>
      </div>
    </div>

    <div class="ctrl-row">
      <span>{{ fmt(cur) }} / {{ fmt(dur) }}</span>
      <div class="nav-btns">
        <button class="btn btn-ghost btn-sm" :disabled="!prevEp" @click="prevEp && switchEp(prevEp)">‹ Prev</button>
        <button class="btn btn-primary btn-sm" :disabled="!nextEp" @click="nextEp && switchEp(nextEp)">Next Ep ›</button>
      </div>
    </div>

    <div class="section-head"><h3>Episodes</h3></div>
    <div class="eps">
      <button
        v-for="e in drama.episodes" :key="e.id"
        class="ep" :class="{ locked: !e.accessible, current: e.id === currentEp?.id }"
        @click="switchEp(e)"
      >{{ e.ep_index }}<span v-if="!e.accessible" class="lk">🔒</span></button>
    </div>
  </div>
  <div v-else class="empty">Loading…</div>
</template>

<style scoped>
.player-page { padding-top: 0; }
.topbar {
  position: sticky; top: 0; z-index: 10;
  display: flex; align-items: center; gap: 10px;
  padding: 10px 12px;
  background: rgba(13,13,18,.9);
  backdrop-filter: blur(10px);
}
.back { font-size: 26px; color: var(--text); }
.title { flex: 1; min-width: 0; }
.t { font-weight: 700; font-size: 14.5px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ep-label { color: var(--muted); font-size: 11px; margin-top: 1px; }
.video-wrap {
  position: relative;
  background: #000;
  aspect-ratio: 16 / 9;
}
.video-wrap video { width: 100%; height: 100%; display: block; background: #000; }
.placeholder { position: absolute; inset: 0; }
.placeholder img { width: 100%; height: 100%; object-fit: cover; opacity: .4; }
.lock-overlay {
  position: absolute; inset: 0;
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px;
  background: rgba(0,0,0,.55);
}
.lock-icon { font-size: 40px; }
.lock-text { font-size: 14px; font-weight: 600; }
.lock-actions { display: flex; gap: 10px; margin-top: 6px; }
.ctrl-row {
  display: flex; justify-content: space-between; align-items: center;
  color: var(--muted); font-size: 12px; padding: 10px 2px;
}
.nav-btns { display: flex; gap: 8px; }
.eps { display: grid; grid-template-columns: repeat(6, 1fr); gap: 8px; }
.ep {
  position: relative;
  background: var(--bg-elev);
  border-radius: 9px;
  padding: 9px 2px;
  font-weight: 800;
  font-size: 13px;
}
.ep .lk { font-size: 9px; margin-left: 2px; }
.ep.locked { color: var(--gold); }
.ep.current { outline: 2px solid var(--accent); }
</style>
