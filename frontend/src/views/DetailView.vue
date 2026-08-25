<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { store, refreshMe } from '../store'

const route = useRoute()
const router = useRouter()
const drama = ref(null)
const error = ref('')

const resumeEp = computed(() => {
  if (!drama.value) return null
  const p = drama.value.progress
  if (p && p.position_sec > 5 && p.position_sec < (p.duration_sec || Infinity) - 10) {
    return drama.value.episodes.find(e => e.id === p.episode_id) || null
  }
  return null
})

const firstLocked = computed(() => {
  if (!drama.value) return null
  return drama.value.episodes.find(e => !e.accessible) || null
})

onMounted(load)

async function load() {
  try {
    drama.value = await api.drama(route.params.id)
  } catch (e) {
    error.value = e.message
  }
}

function watch(ep) {
  router.push(`/player/${drama.value.id}/${ep.id}`)
}

async function toggle(kind) {
  const map = { favorite: 'is_favorite', like: 'is_liked', follow: 'is_followed' }
  const r = await api[kind](drama.value.id)
  drama.value[map[kind]] = r[{ favorite: 'favorited', like: 'liked', follow: 'followed' }[kind]]
  if (kind === 'like') load()
}

async function share() {
  if (navigator.share) {
    try { await navigator.share({ title: drama.value.title }) } catch {}
  } else {
    try { await navigator.clipboard.writeText(location.href) } catch {}
    window.$toast('Link copied')
  }
  try { await api.shareProgress() } catch {}
  window.$toast('Share reward +1 today')
}
</script>

<template>
  <div class="page">
    <button class="back" @click="$router.back()">‹</button>
    <div v-if="drama" class="wrap">
      <div class="hero">
        <img :src="drama.cover" alt="" />
        <div class="info">
          <h2>{{ drama.title }}</h2>
          <div class="tags">
            <span class="badge">{{ drama.category_name || 'Drama' }}</span>
            <span v-if="drama.is_completed" class="badge badge-new">Completed</span>
            <span class="badge">👁 {{ drama.views.toLocaleString() }}</span>
          </div>
          <div class="desc">{{ drama.description }}</div>
        </div>
      </div>

      <div class="actions">
        <button class="btn btn-ghost" @click="toggle('follow')">{{ drama.is_followed ? '✓ Following' : '+ Follow' }}</button>
        <button class="btn btn-ghost" @click="toggle('like')">{{ drama.is_liked ? '❤️ Liked' : '🤍 Like' }}</button>
        <button class="btn btn-ghost" @click="toggle('favorite')">{{ drama.is_favorite ? '⭐ Favorited' : '☆ Favorite' }}</button>
        <button class="btn btn-ghost" @click="share">↗ Share</button>
      </div>

      <button v-if="resumeEp" class="continue btn-primary" @click="watch(resumeEp)">
        ▶ Continue · Ep {{ resumeEp.ep_index }} ({{ Math.floor(resumeEp.duration_sec / 60) }}:{{ String(resumeEp.duration_sec % 60).padStart(2, '0') }})
      </button>

      <div class="section-head">
        <h3>Episodes <span class="count">{{ drama.episodes.length }}</span></h3>
      </div>
      <div class="eps">
        <button
          v-for="e in drama.episodes" :key="e.id"
          class="ep" :class="{ locked: !e.accessible, current: resumeEp && e.id === resumeEp.id }"
          @click="watch(e)"
        >
          <span class="n">{{ e.ep_index }}</span>
          <span v-if="e.free" class="s">Free</span>
          <span v-else-if="e.accessible" class="s ok">✓</span>
          <span v-else class="s lock">🔒</span>
        </button>
      </div>

      <div v-if="firstLocked" class="unlock-tip card">
        <span>🔒 Ep {{ firstLocked.ep_index }}+ need {{ firstLocked.price_coins }} coins each</span>
        <router-link to="/recharge" class="btn btn-gold btn-sm">Top Up</router-link>
      </div>
    </div>
    <div v-else-if="error" class="empty">{{ error }}</div>
    <div v-else class="skel-wrap">
      <div class="skel-hero">
        <div class="skel-poster"></div>
        <div class="skel-lines"><div class="skel-line" style="width:70%"></div><div class="skel-line" style="width:45%"></div><div class="skel-line" style="width:90%"></div></div>
      </div>
      <div class="skel-eps"><div v-for="i in 5" :key="i" class="skel-ep"></div></div>
    </div>
  </div>
</template>

<style scoped>
.back { position: fixed; top: 10px; left: 12px; z-index: 20; font-size: 28px; color: #fff; text-shadow: 0 1px 8px rgba(0,0,0,.8); }
.hero { display: flex; gap: 14px; margin-top: 18px; }
.hero img { width: 118px; border-radius: 12px; aspect-ratio: 3/4; object-fit: cover; }
.info h2 { margin: 0 0 8px; font-size: 20px; }
.tags { display: flex; gap: 6px; flex-wrap: wrap; }
.desc { color: var(--muted); font-size: 12.5px; margin-top: 10px; display: -webkit-box; -webkit-line-clamp: 4; -webkit-box-orient: vertical; overflow: hidden; }
.actions { display: flex; gap: 10px; margin: 16px 0 6px; }
.actions .btn { flex: 1; }
.continue { width: 100%; margin: 10px 0; padding: 13px; border-radius: 14px; font-size: 15px; }
.count { color: var(--muted); font-size: 13px; font-weight: 500; }
.eps { display: grid; grid-template-columns: repeat(5, 1fr); gap: 8px; }
.ep {
  position: relative;
  background: var(--bg-elev);
  border-radius: 10px;
  padding: 10px 4px 7px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
}
.ep .n { font-weight: 800; font-size: 14px; }
.ep .s { font-size: 9px; color: var(--green); }
.ep .s.lock { color: var(--gold); }
.ep.locked { opacity: .85; }
.ep.current { outline: 2px solid var(--accent); }
.unlock-tip {
  margin-top: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  color: var(--muted);
}
/* skeletons */
.skel-poster { width: 118px; height: 157px; border-radius: 12px; background: var(--bg-elev); animation: pulse 1.2s ease-in-out infinite; }
.skel-hero { display: flex; gap: 14px; margin-top: 18px; }
.skel-lines { flex: 1; display: flex; flex-direction: column; gap: 10px; padding-top: 8px; }
.skel-line { height: 14px; border-radius: 6px; background: var(--bg-elev); animation: pulse 1.2s ease-in-out infinite; }
.skel-eps { display: grid; grid-template-columns: repeat(5, 1fr); gap: 8px; margin-top: 24px; }
.skel-ep { height: 48px; border-radius: 10px; background: var(--bg-elev); animation: pulse 1.2s ease-in-out infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: .45; } }
</style>
