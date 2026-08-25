<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import DramaCard from '../components/DramaCard.vue'

const list = ref(null)
onMounted(async () => {
  list.value = await api.favorites()
})
</script>

<template>
  <div class="page">
    <div class="section-head"><h3>📚 My Playlist</h3></div>
    <div v-if="list && list.length" class="grid">
      <DramaCard v-for="d in list" :key="d.id" :drama="d" />
    </div>
    <div v-else-if="list" class="empty">
      No favorites yet.<br />Open a drama and tap "☆ Favorite" to add it here.
    </div>
    <div v-else class="empty">Loading…</div>
  </div>
</template>

<style scoped>
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
</style>
