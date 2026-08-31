<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { store, fmtCoins } from '../store'
import DramaCard from '../components/DramaCard.vue'

const router = useRouter()
const data = ref(null)
const error = ref('')
const bannerEl = ref(null)
const bannerIdx = ref(0)
let autoTimer = null
let resumeTimer = null

const bannerCount = computed(() => (data.value?.banners || []).length)

function slideTo(i) {
  const el = bannerEl.value
  if (!el) return
  bannerIdx.value = ((i % bannerCount.value) + bannerCount.value) % bannerCount.value
  el.scrollTo({ left: bannerIdx.value * el.clientWidth, behavior: 'smooth' })
}
function startAuto() {
  stopAuto()
  autoTimer = setInterval(() => { if (bannerCount.value > 1) slideTo(bannerIdx.value + 1) }, 3500)
}
function stopAuto() { clearInterval(autoTimer) }
// banner jump: custom link wins, then drama -> straight into its player (ep 0 = first)
function openBanner(b) {
  if (b.link) {
    if (b.link.startsWith('#')) location.hash = b.link.slice(1)
    else location.href = b.link
    return
  }
  if (b.drama_id) router.push(`/player/${b.drama_id}/0`)
}
function onTouchStart() { stopAuto(); clearTimeout(resumeTimer) }
function onTouchEnd() { resumeTimer = setTimeout(startAuto, 4000) }

onMounted(async () => {
  try {
    data.value = await api.home()
    if ((data.value?.banners || []).length > 1) startAuto()
  } catch (e) { error.value = e.message }
})
onBeforeUnmount(() => { stopAuto(); clearTimeout(resumeTimer) })
</script>

<template>
  <div>
    <div class="search-bar">
      <router-link to="/search" class="search-input">
        <span class="icon">🔍</span>
        <span>Search dramas…</span>
      </router-link>
      <router-link to="/wallet" class="coins">
        <span class="coin">{{ store.user ? fmtCoins(store.user.coins) : '' }}</span>
      </router-link>
    </div>

    <nav class="top-tabs">
      <span class="active">Home</span>
      <router-link to="/charts" tag="span">Charts</router-link>
      <router-link to="/list" tag="span">List</router-link>
      <router-link to="/list?category=1" tag="span">Classification</router-link>
      <router-link to="/list?completed=1" tag="span">Completed</router-link>
    </nav>

    <div class="page" v-if="data">
      <!-- banner carousel -->
      <div class="banner-wrap">
        <div class="banners" ref="bannerEl" @touchstart.passive="onTouchStart" @touchend.passive="onTouchEnd">
          <div
            v-for="b in data.banners"
            :key="b.image + (b.link || '') + b.drama_id"
            class="banner"
            @click="openBanner(b)"
          >
            <img :src="b.image" alt="" />
          </div>
        </div>
        <div v-if="bannerCount > 1" class="dots">
          <span
            v-for="i in bannerCount" :key="i"
            class="dot" :class="{ on: bannerIdx === i - 1 }"
            @click="slideTo(i - 1)"
          ></span>
        </div>
      </div>

      <div class="section-head">
        <h3>⭐ Featured</h3>
        <router-link class="more" to="/list?featured=1">See more ›</router-link>
      </div>
      <div class="grid">
        <DramaCard v-for="d in data.featured" :key="d.id" :drama="d" />
      </div>

      <div class="section-head"><h3>🔥 Trending Now</h3></div>
      <div class="grid">
        <DramaCard v-for="d in data.hot" :key="d.id" :drama="d" />
      </div>

      <div class="section-head"><h3>🆕 New Release</h3>
        <router-link class="more" to="/list">See more ›</router-link>
      </div>
      <div class="grid">
        <DramaCard v-for="d in data.new_releases" :key="d.id" :drama="d" />
      </div>

      <template v-for="ch in data.channels" :key="ch.id">
        <div class="section-head">
          <h3>📺 {{ ch.name }}</h3>
        </div>
        <div v-if="ch.dramas && ch.dramas.length" class="grid">
          <DramaCard v-for="d in ch.dramas" :key="d.id" :drama="d" />
        </div>
        <div v-else class="empty">No dramas in this channel yet</div>
      </template>
    </div>
    <div v-else-if="error" class="empty">{{ error }}</div>
    <div v-else class="empty">Loading…</div>
  </div>
</template>

<style scoped>
.banner-wrap { position: relative; }
.dots {
  position: absolute;
  left: 0; right: 0; bottom: 8px;
  display: flex; justify-content: center; gap: 5px;
}
.dot {
  width: 6px; height: 6px; border-radius: 999px;
  background: rgba(255, 255, 255, .4);
  transition: all .2s;
}
.dot.on { background: #fff; width: 14px; }
.search-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 12px 14px 0;
  max-width: 560px;
  margin: 0 auto;
}
.search-input {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--bg-elev);
  border-radius: 999px;
  padding: 10px 16px;
  color: var(--muted);
  font-size: 13px;
}
.coins { flex-shrink: 0; }
.banners {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  border-radius: 14px;
  scrollbar-width: none;
}
.banners::-webkit-scrollbar { display: none; }
.banner {
  flex: 0 0 100%;
  scroll-snap-align: center;
  border-radius: 14px;
  overflow: hidden;
  aspect-ratio: 16 / 7.5;
  cursor: pointer;
}
.banner img { width: 100%; height: 100%; object-fit: cover; display: block; }
.grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}
</style>
