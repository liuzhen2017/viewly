<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { store, refreshMe, fmtCoins } from '../store'
import AdPlayer from '../components/AdPlayer.vue'

const route = useRoute()
const router = useRouter()

const videoEl = ref(null)
const drama = ref(null)
const currentEp = ref(null)
const playing = ref(false)
const showUnlock = ref(false)
const unlockBusy = ref(false)
const showAd = ref(false)
const cur = ref(0)
const dur = ref(0)
const isPortrait = ref(false)
let touchY0 = null
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
      // stop and detach any previous episode's stream so the unlock
      // overlay is guaranteed to be visible on sequential navigation
      const v = videoEl.value
      if (v) {
        try { v.pause(); v.removeAttribute('src'); v.load() } catch {}
      }
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
      // coins short: try a rewarded ad first, only send to recharge if no ads
      tryAdForUnlock()
    } else {
      window.$toast(e.message)
    }
  } finally {
    unlockBusy.value = false
  }
}

// rewarded-ad loop: watch ad -> credit coins -> retry unlock automatically
function adAvailable() {
  return store.adConfig && store.adConfig.rewarded_ad_mode === 'client'
}
function tryAdForUnlock() {
  if (!adAvailable()) {
    window.$toast('Not enough coins')
    router.push('/recharge')
    return
  }
  showUnlock.value = false
  showAd.value = true
}
async function onAdDone() {
  showAd.value = false
  try {
    const r = await api.watchAdComplete()
    await refreshMe()
    window.$toast(`+${r.coins} coins — unlocking…`)
    unlockBusy.value = true
    await confirmUnlock()
  } catch (e) {
    if (e.code === 5003) {
      window.$toast('Daily ad limit reached')
      router.push('/recharge')
    } else {
      window.$toast(e.message)
      showUnlock.value = true
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
// vertical swipe in the video area: up = next ep, down = prev ep
function onTouchedVideo(e) {
  if (e.touches.length) touchY0 = e.touches[0].clientY
}
function onReleasedVideo(e) {
  if (touchY0 === null) return
  const dy = (e.changedTouches[0].clientY || 0) - touchY0
  touchY0 = null
  if (dy < -56 && nextEp.value) switchEp(nextEp.value)
  else if (dy > 56 && prevEp.value) switchEp(prevEp.value)
}
function onVideoMeta() {
  const v = videoEl.value
  if (v && v.videoHeight > v.videoWidth) isPortrait.value = true
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
      <button class="back" @click="$router.replace(`/drama/${drama.id}`)">‹</button>
      <div class="title">
        <div class="t">{{ drama.title }}</div>
        <div class="ep-label" v-if="currentEp">Ep {{ currentEp.ep_index }} · {{ currentEp.title }}</div>
      </div>
      <router-link to="/wallet" class="coin">{{ fmtCoins(store.user?.coins ?? 0) }}</router-link>
    </div>

    <div class="video-wrap" :class="{ portrait: isPortrait }">
      <video
        ref="videoEl"
        playsinline
        controls
        @play="playing = true"
        @pause="report"
        @timeupdate="onTime"
        @ended="onEnded"
        @loadedmetadata="onVideoMeta"
        @touchstart.passive="onTouchedVideo"
        @touchend.passive="onReleasedVideo"
      ></video>
      <div v-if="!videoEl?.src || showUnlock" class="placeholder">
        <img v-if="currentEp" :src="drama.cover" alt="" />
        <div class="lock-overlay" v-if="showUnlock">
          <div class="lock-icon">🔒</div>
          <div class="lock-text">Ep {{ currentEp?.ep_index }} requires {{ currentEp?.price_coins }} coins</div>
          <div class="lock-actions">
            <button class="btn btn-primary" :disabled="unlockBusy" @click="confirmUnlock">
              {{ unlockBusy ? '⏳ Unlocking…' : 'Unlock · ' + currentEp?.price_coins + ' 🪙' }}
            </button>
            <button
              v-if="adAvailable() && (store.user?.coins ?? 0) < (currentEp?.price_coins || 0)"
              class="btn btn-gold" @click="tryAdForUnlock"
            >▶ Watch Ad +{{ store.adConfig?.rewarded_ad_coins || 50 }} 🪙</button>
            <router-link to="/recharge" class="btn btn-ghost">Top Up</router-link>
          </div>
        </div>
      </div>
    </div>

    <div class="ctrl-row">
      <span>{{ fmt(cur) }} / {{ fmt(dur) }}</span>
    </div>
    <div class="ep-nav">
      <button class="ep-nav-btn" :disabled="!prevEp" @click="prevEp && switchEp(prevEp)">
        <span class="dir">‹</span>
        <span class="lbl">{{ prevEp ? 'Ep ' + prevEp.ep_index : 'First' }}</span>
      </button>
      <button class="ep-nav-btn primary" :disabled="!nextEp" @click="nextEp && switchEp(nextEp)">
        <span class="lbl">{{ nextEp ? 'Ep ' + nextEp.ep_index : 'Last' }}</span>
        <span class="dir">›</span>
      </button>
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
  <div v-else class="page">
    <div class="skel-video"></div>
    <div class="skel-bar" style="width:40%"></div>
    <div class="skel-grid">
      <div v-for="i in 6" :key="i" class="skel-ep"></div>
    </div>
  </div>
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
.video-wrap.portrait {
  aspect-ratio: auto;
  height: 64vh;
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
  color: var(--muted); font-size: 12px; padding: 10px 2px 6px;
}
.ep-nav { display: flex; gap: 10px; margin-bottom: 14px; }
.ep-nav-btn {
  flex: 1;
  display: flex; align-items: center; justify-content: center; gap: 8px;
  padding: 13px 0;
  border-radius: 12px;
  background: var(--bg-elev);
  font-size: 15px;
  font-weight: 800;
}
.ep-nav-btn .dir { font-size: 18px; opacity: .8; }
.ep-nav-btn.primary { background: linear-gradient(135deg, var(--accent), var(--accent-2)); color: #fff; }
.ep-nav-btn[disabled] { opacity: .4; }
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
.skel-video { aspect-ratio: 16/9; background: var(--bg-elev); border-radius: 0; animation: pulse 1.2s ease-in-out infinite; }
.skel-bar { height: 14px; background: var(--bg-elev); border-radius: 6px; margin: 12px 2px; animation: pulse 1.2s ease-in-out infinite; }
.skel-grid { display: grid; grid-template-columns: repeat(6, 1fr); gap: 8px; }
.skel-ep { height: 38px; background: var(--bg-elev); border-radius: 9px; animation: pulse 1.2s ease-in-out infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: .45; } }
</style>
