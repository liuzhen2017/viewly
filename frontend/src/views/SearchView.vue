<script setup>
import { ref } from 'vue'
import { api } from '../api'
import DramaCard from '../components/DramaCard.vue'

const kw = ref('')
const list = ref(null)
const searched = ref(false)

async function doSearch() {
  if (!kw.value.trim()) return
  list.value = await api.search(kw.value.trim())
  searched.value = true
}
</script>

<template>
  <div class="page">
    <div class="search-row">
      <button class="back" @click="$router.back()">‹</button>
      <input
        v-model="kw"
        placeholder="Search dramas…"
        enterkeyhint="search"
        @keyup.enter="doSearch"
      />
      <button class="btn btn-primary" @click="doSearch">Search</button>
    </div>
    <div v-if="list && list.length" class="grid">
      <DramaCard v-for="d in list" :key="d.id" :drama="d" />
    </div>
    <div v-else-if="searched" class="empty">No results for “{{ kw }}”</div>
    <div v-else class="empty">Type a title to search</div>
  </div>
</template>

<style scoped>
.search-row { display: flex; gap: 10px; align-items: center; margin-bottom: 16px; }
.back { font-size: 26px; color: var(--muted); padding: 0 4px; }
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
</style>
