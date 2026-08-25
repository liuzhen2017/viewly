<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { store, fmtCoins } from '../store'
import DramaCard from '../components/DramaCard.vue'

const data = ref(null)
const error = ref('')

onMounted(async () => {
  try { data.value = await api.home() } catch (e) { error.value = e.message }
})
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
      <div class="banners">
        <router-link
          v-for="b in data.banners"
          :key="b.image"
          :to="b.drama_id ? `/drama/${b.drama_id}` : '/#'"
          class="banner"
        >
          <img :src="b.image" alt="" />
        </router-link>
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
}
.banner img { width: 100%; height: 100%; object-fit: cover; display: block; }
.grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}
</style>
