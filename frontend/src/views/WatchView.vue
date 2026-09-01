<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { store, fmtCoins } from '../store'

const router = useRouter()

// TikTok-style vertical feed: full-viewport cards with scroll-snap; only the
// visible card plays (IntersectionObserver pauses the rest).
const feed = ref([])
const page = ref(1)
const loading = ref(false)
let observer = null
let videos = []

onMounted(async () => {
  await loadMore()
  await nextTick()
  observer = new IntersectionObserver((entries) => {
    entries.forEach(en => {
      const v = en.target.querySelector('video')
      if (!v) return
      if (en.isIntersecting && en.intersectionRatio > 0.6) {
        v.play().catch(() => {})
        const ep = feed.value.find(f => 'ep' + f.episode_id === en.target.dataset.key)
        if (ep && !v.src) v.src = ep.video_url || ''
      } else {
        v.pause()
      }
    })
  }, { threshold: [0, 0.6, 1] })
  document.querySelectorAll('.feed-item').forEach(el => observer.observe(el))
  window.addEventListener('scroll', onScroll, { passive: true })
})

onBeforeUnmount(() => {
  observer?.disconnect()
  window.removeEventListener('scroll', onScroll)
})

async function loadMore() {
  if (loading.value) return
  loading.value = true
  const more = await api.feed(page.value)
  feed.value.push(...more)
  page.value++
  loading.value = false
}

let ticking = false
function onScroll() {
  if (ticking) return
  ticking = true
  requestAnimationFrame(() => {
    if (innerHeight + scrollY > document.body.scrollHeight - 400) loadMore()
    ticking = false
  })
}

function openPlayer(f) {
  router.push(`/player/${f.drama_id}/${f.episode_id}`)
}

// Prev/Next from the feed card: lazily fetch the drama's episode list once,
// then jump straight into the fullscreen player at the target episode.
const dramaCache = {}
async function openEpisode(f, dir) {
  try {
    let d = dramaCache[f.drama_id]
    if (!d) {
      d = await api.drama(f.drama_id)
      dramaCache[f.drama_id] = d
    }
    const eps = (d.episodes || []).slice().sort((a, b) => a.ep_index - b.ep_index)
    const idx = eps.findIndex(e => e.ep_index === f.ep_index)
    const target = eps[idx + dir]
    if (!target) {
      window.$toast(dir > 0 ? 'Already the last episode' : 'Already Ep 1')
      return
    }
    router.push(`/player/${f.drama_id}/${target.id}`)
  } catch (e) {
    window.$toast(e.message)
  }
}
</script>

<template>
  <div class="feed-wrap">
    <div
      v-for="f in feed" :key="f.episode_id" class="feed-item" :data-key="'ep' + f.episode_id"
      @click="openPlayer(f)"
    >
      <img class="bg" :src="f.cover" alt="" />
      <!-- preview only: pointer-events off so in-app webviews can't hijack taps -->
      <video playsinline webkit-playsinline x5-playsinline loop muted></video>
      <div class="tap-hint">
        <div class="tap-circle">▶</div>
      </div>
      <div class="meta">
        <div class="drama-title">
          <b>{{ f.drama_title }}</b> <span class="badge">{{ f.category }}</span>
        </div>
        <div class="ep-line">
          <button class="ep-btn" :disabled="f.ep_index <= 1" @click.stop="openEpisode(f, -1)">‹ Prev</button>
          <span class="ep-num">Ep {{ f.ep_index }} · 👁 {{ f.views.toLocaleString() }}</span>
          <button class="ep-btn" @click.stop="openEpisode(f, 1)">Next ›</button>
        </div>
      </div>
    </div>
    <div v-if="!feed.length" class="empty" style="padding-top:120px">Loading feed…</div>
  </div>
</template>

<style scoped>
.feed-wrap {
  height: 100vh;
  overflow-y: auto;
  scroll-snap-type: y mandatory;
  scrollbar-width: none;
}
.feed-wrap::-webkit-scrollbar { display: none; }
.feed-item {
  position: relative;
  height: 100vh;
  scroll-snap-align: start;
  scroll-snap-stop: always;
  overflow: hidden;
}
.feed-item .bg { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; filter: brightness(.55); }
.feed-item video { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: contain; background: transparent; pointer-events: none; }
.tap-hint {
  position: absolute; inset: 0;
  display: flex; align-items: center; justify-content: center;
  pointer-events: none;
}
.tap-circle {
  width: 74px; height: 74px; border-radius: 50%;
  background: rgba(255, 90, 60, .92);
  display: flex; align-items: center; justify-content: center;
  font-size: 30px; color: #fff;
  box-shadow: 0 6px 24px rgba(0, 0, 0, .45);
}
.feed-item { cursor: pointer; }
.meta {
  position: absolute;
  left: 16px; right: 16px; bottom: calc(var(--nav-h) + env(safe-area-inset-bottom, 0px) + 22px);
  display: flex; flex-direction: column; gap: 8px;
}
.drama-title { font-size: 16px; display: flex; align-items: center; gap: 8px; }
.ep-line {
  color: rgba(255,255,255,.75); font-size: 12.5px;
  display: flex; align-items: center; gap: 10px;
}
.ep-btn {
  padding: 9px 18px; border-radius: 999px;
  background: rgba(255, 90, 60, .9); color: #fff;
  font-size: 14px; font-weight: 600;
  border: none; cursor: pointer;
  box-shadow: 0 4px 14px rgba(0, 0, 0, .35);
}
.ep-btn:disabled { background: rgba(255,255,255,.18); color: rgba(255,255,255,.5); }
.ep-btn:not(:disabled):active { transform: scale(.94); }
.ep-num { min-width: 86px; text-align: center; }
.row { display: flex; gap: 10px; margin-top: 4px; }
.play-btn { padding: 11px 28px; }
</style>
