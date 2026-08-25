<script setup>
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import BottomNav from './components/BottomNav.vue'

const route = useRoute()
const toast = ref('')
let toastTimer = null
window.$toast = (msg) => {
  toast.value = msg
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = ''), 1800)
}
</script>

<template>
  <router-view v-slot="{ Component }">
    <component :is="Component" />
  </router-view>
  <BottomNav v-if="route.meta.tab" />
  <div v-if="toast" class="toast">{{ toast }}</div>
</template>
