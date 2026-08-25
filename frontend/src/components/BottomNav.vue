<script setup>
import { useRoute } from 'vue-router'
const route = useRoute()
const tabs = [
  { key: 'home', to: '/', icon: '🏠', label: 'Home' },
  { key: 'watch', to: '/watch', icon: '▶️', label: 'Watch' },
  { key: 'rewards', to: '/rewards', icon: '🎁', label: 'Rewards' },
  { key: 'playlist', to: '/playlist', icon: '📚', label: 'Playlist' },
  { key: 'profile', to: '/profile', icon: '👤', label: 'Profile' },
]
</script>

<template>
  <nav class="bottom-nav">
    <router-link v-for="t in tabs" :key="t.key" :to="t.to" :class="{ active: route.meta.tab === t.key }">
      <span class="ico">{{ t.icon }}</span>
      <span>{{ t.label }}</span>
    </router-link>
  </nav>
</template>
