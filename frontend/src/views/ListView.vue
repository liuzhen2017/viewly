<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import DramaCard from '../components/DramaCard.vue'

const route = useRoute()
const list = ref([])
const cats = ref([])
const activeCat = ref(route.query.category || '')
const completed = ref(route.query.completed === '1')
const loading = ref(false)
const total = ref(0)

async function load() {
  loading.value = true
  const params = { size: 30 }
  if (activeCat.value) params.category_id = activeCat.value
  if (completed.value) params.completed = 1
  const r = await api.dramas(params)
  list.value = r.list || []
  total.value = r.total || 0
  loading.value = false
}

onMounted(async () => {
  cats.value = await api.categories()
  await load()
})
watch([activeCat, completed], load)
watch(() => route.query, () => {
  activeCat.value = route.query.category || ''
  completed.value = route.query.completed === '1'
})
</script>

<template>
  <div>
    <nav class="top-tabs">
      <router-link to="/" tag="span">Home</router-link>
      <router-link to="/charts" tag="span">Charts</router-link>
      <span class="active">List</span>
    </nav>
    <div class="page">
      <div class="chips">
        <button class="chip" :class="{ on: !activeCat && !completed }" @click="activeCat = ''; completed = false">All</button>
        <button
          v-for="c in cats" :key="c.id"
          class="chip" :class="{ on: String(c.id) === activeCat }"
          @click="activeCat = String(c.id); completed = false"
        >{{ c.name }}</button>
        <button class="chip" :class="{ on: completed }" @click="completed = !completed; if (completed) activeCat = ''">✅ Completed</button>
      </div>
      <div v-if="list.length" class="grid">
        <DramaCard v-for="d in list" :key="d.id" :drama="d" />
      </div>
      <div v-else-if="!loading" class="empty">No dramas found</div>
    </div>
  </div>
</template>

<style scoped>
.chips {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 4px 0 14px;
  scrollbar-width: none;
}
.chips::-webkit-scrollbar { display: none; }
.chip {
  flex-shrink: 0;
  padding: 7px 14px;
  border-radius: 999px;
  background: var(--bg-elev);
  font-size: 12.5px;
  font-weight: 600;
  color: var(--muted);
}
.chip.on { background: linear-gradient(135deg, var(--accent), var(--accent-2)); color: #fff; }
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
</style>
