<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import DramaCard from '../components/DramaCard.vue'

const list = ref([])
onMounted(async () => {
  list.value = await api.dramas({ sort: 'views', size: 30 })
})
</script>

<template>
  <div>
    <nav class="top-tabs">
      <router-link to="/" tag="span">Home</router-link>
      <span class="active">Charts</span>
      <router-link to="/list" tag="span">List</router-link>
    </nav>
    <div class="page">
      <div class="section-head"><h3>🔥 Top Charts</h3></div>
      <div class="rank" v-for="(d, i) in list" :key="d.id">
        <span class="no" :class="{ top: i < 3 }">{{ i + 1 }}</span>
        <router-link :to="`/drama/${d.id}`" class="row">
          <img :src="d.cover" class="thumb" alt="" />
          <div class="info">
            <div class="t">{{ d.title }}</div>
            <div class="m">👁 {{ d.views.toLocaleString() }} views · {{ d.episodes }} eps</div>
          </div>
        </router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.rank { display: flex; align-items: center; gap: 12px; padding: 8px 0; }
.no { width: 22px; text-align: center; font-size: 16px; font-weight: 900; color: var(--muted); font-style: italic; }
.no.top { color: var(--accent); }
.row { flex: 1; display: flex; gap: 12px; min-width: 0; }
.thumb { width: 56px; height: 74px; border-radius: 8px; object-fit: cover; }
.info { min-width: 0; align-self: center; }
.t { font-weight: 700; font-size: 14px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.m { color: var(--muted); font-size: 11.5px; margin-top: 4px; }
</style>
