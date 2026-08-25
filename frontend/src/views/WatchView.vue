<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { api } from '../api'
import { store, fmtCoins } from '../store'

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

async function goWatch(f) {
  try {
    const r = await api.play(f.episode_id)
    const i = feed.value.findIndex(x => x.episode_id === f.episode_id)
    if (i >= 0) {
      const card = document.querySelector(`[data-key="ep${f.episode_id}"] video`)
      if (card) { card.src = r.video_url; card.play().catch(() => {}) }
    }
  } catch (e) {
    window.$toast(e.code === 402 ? `🔒 ${f.price_coins} coins to unlock` : e.message)
  }
}
</script>

<template>
  <div class="feed-wrap">
    <div
      v-for="f in feed" :key="f.episode_id" class="feed-item" :data-key="'ep' + f.episode_id"
    >
      <img class="bg" :src="f.cover" alt="" />
      <video playsinline loop muted></video>
      <div class="meta">
        <router-link :to="`/drama/${f.drama_id}`" class="drama-title">
          <b>{{ f.drama_title }}</b> <span class="badge">{{ f.category }}</span>
        </router-link>
        <div class="ep-line">Ep {{ f.ep_index }} · 👁 {{ f.views.toLocaleString() }}</div>
        <div class="row">
          <button class="play-btn btn btn-primary" @click="goWatch(f)">▶ Watch</button>
          <router-link v-if="!f.accessible" to="/recharge" class="btn btn-gold btn-sm">🔒 {{ f.price_coins }} 🪙</router-link>
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
.feed-item video { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: contain; background: transparent; }
.meta {
  position: absolute;
  left: 16px; right: 16px; bottom: calc(var(--nav-h) + env(safe-area-inset-bottom, 0px) + 22px);
  display: flex; flex-direction: column; gap: 8px;
}
.drama-title { font-size: 16px; display: flex; align-items: center; gap: 8px; }
.ep-line { color: rgba(255,255,255,.75); font-size: 12.5px; }
.row { display: flex; gap: 10px; margin-top: 4px; }
.play-btn { padding: 11px 28px; }
</style>
