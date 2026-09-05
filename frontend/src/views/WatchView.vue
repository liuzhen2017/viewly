<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { store, fmtCoins } from '../store'

// TikTok-style vertical feed: full-viewport cards with scroll-snap. Visible
// cards autoplay muted previews (when the episode is accessible); tapping a
// card jumps into the fullscreen player with sound. A List button opens the
// drama's episode picker (paginated grid) straight from the feed.
const router = useRouter()
const feed = ref([])
const page = ref(1)
const loading = ref(false)
let observer = null

onMounted(async () => {
  await loadMore()
  await nextTick()
  observer = new IntersectionObserver((entries) => {
    entries.forEach(en => {
      const v = en.target.querySelector('video')
      if (!v) return
      if (en.isIntersecting && en.intersectionRatio > 0.6) {
        const ep = feed.value.find(f => 'ep' + f.episode_id === en.target.dataset.key)
        if (ep && !v.src && ep.video_url) v.src = ep.video_url
        v.play().catch(() => {})
        // warm the next card so swiping up starts instantly
        const next = en.target.nextElementSibling?.querySelector('video')
        if (next && !next.src) {
          const nk = en.target.nextElementSibling.dataset.key.slice(2)
          const nf = feed.value.find(f => String(f.episode_id) === nk)
          if (nf?.video_url) next.preload = 'auto', next.src = nf.video_url, next.pause()
        }
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
  await nextTick()
  document.querySelectorAll('.feed-item:not([data-obs])').forEach(el => {
    el.dataset.obs = '1'
    observer?.observe(el)
  })
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

// ---- episode picker drawer (competitor-style List button) ----
const picker = ref(null)     // feed item the picker was opened for
const pickerEps = ref([])
const pickerPage = ref(0)    // 25 episodes per page, like the reference app
const pickerOpen = ref(false)
const dramaCache = {}

async function openPicker(f) {
  picker.value = f
  pickerOpen.value = true
  try {
    let d = dramaCache[f.drama_id]
    if (!d) {
      d = await api.drama(f.drama_id)
      dramaCache[f.drama_id] = d
    }
    pickerEps.value = (d.episodes || []).slice().sort((a, b) => a.ep_index - b.ep_index)
    pickerPage.value = Math.floor((f.ep_index - 1) / 25)
  } catch (e) {
    window.$toast(e.message)
  }
}
const pickerRanges = computed(() => {
  const n = pickerEps.value.length
  const out = []
  for (let i = 0; i < n; i += 25) out.push([i, Math.min(i + 25, n) - 1])
  return out
})
function pickEp(ep) {
  pickerOpen.value = false
  router.push(`/player/${picker.value.drama_id}/${ep.id}`)
}

// Prev/Next from the feed card: jump straight into the fullscreen player at
// the neighboring episode.
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
      <!-- muted autoplay preview; paid episodes keep the cover only -->
      <video v-if="f.video_url" playsinline webkit-playsinline x5-playsinline loop muted></video>
      <div v-if="!f.video_url" class="lock-badge">🔒 {{ f.price_coins }} 🪙</div>
      <div class="rail">
        <button class="rail-btn" @click.stop="openPicker(f)">
          <span class="rail-ico">☰</span><span class="rail-txt">List</span>
        </button>
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

    <!-- episode picker drawer -->
    <div v-if="pickerOpen" class="sheet-mask" @click="pickerOpen = false">
      <div class="sheet" @click.stop>
        <div class="sheet-head">
          <b>{{ picker?.drama_title }}</b>
          <button class="sheet-x" @click="pickerOpen = false">✕</button>
        </div>
        <div class="sheet-tabs">
          <button
            v-for="(r, i) in pickerRanges" :key="i" class="sheet-tab" :class="{ on: i === pickerPage }"
            @click="pickerPage = i"
          >{{ pickerEps[r[0]].ep_index }} – {{ pickerEps[r[1]].ep_index }}</button>
        </div>
        <div class="sheet-grid">
          <button
            v-for="e in pickerEps.slice(pickerRanges[pickerPage]?.[0] ?? 0, (pickerRanges[pickerPage]?.[1] ?? -1) + 1)"
            :key="e.id" class="sheet-ep" :class="{ cur: picker && e.ep_index === picker.ep_index, free: e.free }"
            @click="pickEp(e)"
          >{{ e.ep_index }}</button>
        </div>
      </div>
    </div>
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
  cursor: pointer;
}
.feed-item .bg { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; filter: brightness(.55); }
.feed-item video { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: contain; background: transparent; pointer-events: none; }
.lock-badge {
  position: absolute; top: 18px; right: 76px;
  padding: 7px 13px; border-radius: 999px;
  background: rgba(0,0,0,.55); color: #ffd76a;
  font-size: 13px; font-weight: 600;
}
.rail {
  position: absolute; right: 12px; bottom: calc(var(--nav-h) + env(safe-area-inset-bottom, 0px) + 120px);
  display: flex; flex-direction: column; gap: 18px;
}
.rail-btn {
  width: 52px; padding: 9px 0 7px; border-radius: 50%;
  background: rgba(0,0,0,.45); color: #fff; border: none;
  display: flex; flex-direction: column; align-items: center; gap: 1px;
  cursor: pointer;
}
.rail-ico { font-size: 21px; line-height: 1; }
.rail-txt { font-size: 10.5px; opacity: .92; }
.rail-btn:active { transform: scale(.92); }
.meta {
  position: absolute;
  left: 16px; right: 80px;
  bottom: calc(var(--nav-h) + env(safe-area-inset-bottom, 0px) + 22px);
  display: flex; flex-direction: column; gap: 8px;
}
.drama-title { font-size: 16px; display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
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

/* episode picker drawer */
.sheet-mask {
  position: fixed; inset: 0; z-index: 60;
  background: rgba(0,0,0,.55);
  display: flex; align-items: flex-end;
}
.sheet {
  width: 100%; max-height: 68vh; overflow-y: auto;
  background: #1c1c26; border-radius: 18px 18px 0 0;
  padding: 16px 16px calc(20px + env(safe-area-inset-bottom, 0px));
}
.sheet-head { display: flex; justify-content: space-between; align-items: center; font-size: 15px; margin-bottom: 12px; }
.sheet-x { background: none; border: none; color: #fff; font-size: 18px; padding: 4px 8px; cursor: pointer; }
.sheet-tabs { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 14px; }
.sheet-tab {
  padding: 7px 14px; border-radius: 999px; border: none; cursor: pointer;
  background: rgba(255,255,255,.1); color: rgba(255,255,255,.8); font-size: 13px;
}
.sheet-tab.on { background: #ff5a3c; color: #fff; font-weight: 600; }
.sheet-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 10px; }
.sheet-ep {
  padding: 12px 0; border-radius: 9px; border: none; cursor: pointer;
  background: rgba(255,255,255,.09); color: #fff; font-size: 15px;
}
.sheet-ep.free { color: #8fe3a2; }
.sheet-ep.cur { background: #ff5a3c; color: #fff; font-weight: 700; }
</style>
