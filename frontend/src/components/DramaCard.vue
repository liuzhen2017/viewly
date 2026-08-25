<script setup>
defineProps({
  drama: { type: Object, required: true },
  wide: Boolean,
})
</script>

<template>
  <router-link :to="`/drama/${drama.id}`" class="drama-card" :class="{ wide }">
    <div class="poster">
      <img :src="drama.cover" :alt="drama.title" loading="lazy" />
      <span v-if="drama.is_completed" class="fin">Completed</span>
      <span v-else class="upd">Updating</span>
      <span class="eps">{{ drama.episodes }} eps</span>
    </div>
    <div class="title">{{ drama.title }}</div>
    <div class="meta">
      <span>👁 {{ drama.views > 9999 ? (drama.views / 1000).toFixed(1) + 'K' : drama.views }}</span>
      <span v-if="drama.is_hot" class="hot">🔥 Hot</span>
    </div>
  </router-link>
</template>

<style scoped>
.drama-card { display: block; min-width: 0; }
.poster {
  position: relative;
  border-radius: 12px;
  overflow: hidden;
  aspect-ratio: 3 / 4;
  background: var(--bg-elev);
}
.poster img { width: 100%; height: 100%; object-fit: cover; display: block; }
.poster .fin, .poster .upd {
  position: absolute;
  left: 6px;
  top: 6px;
  font-size: 9px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 5px;
  background: rgba(0, 0, 0, .55);
  backdrop-filter: blur(4px);
}
.poster .fin { color: var(--green); }
.poster .upd { color: var(--gold); }
.poster .eps {
  position: absolute;
  right: 6px;
  bottom: 6px;
  font-size: 9px;
  padding: 2px 6px;
  border-radius: 5px;
  background: rgba(0, 0, 0, .55);
  color: #fff;
}
.title {
  margin-top: 7px;
  font-size: 12.5px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.meta {
  display: flex;
  justify-content: space-between;
  font-size: 10.5px;
  color: var(--muted);
  margin-top: 2px;
}
.meta .hot { color: var(--accent-2); }
</style>
