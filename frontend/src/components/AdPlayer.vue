<script setup>
import { ref, watch, onBeforeUnmount } from 'vue'

// Rewarded-video overlay. Dev builds simulate a 5s ad; production H5 should
// swap the countdown for a real rewarded SDK (Google Ad Manager web SDK /
// AppLovin MAX web) and emit 'done' from its reward callback.
const props = defineProps({ visible: Boolean })
const emit = defineEmits(['done', 'cancel'])
const countdown = ref(5)
let timer = null

watch(() => props.visible, v => {
  clearInterval(timer)
  if (v) {
    countdown.value = 5
    timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(timer)
        emit('done')
      }
    }, 1000)
  }
})
onBeforeUnmount(() => clearInterval(timer))
</script>

<template>
  <div v-if="visible" class="ad-overlay">
    <div class="ad-box">
      <div class="ad-video">
        <div class="ad-placeholder">Ad playing…</div>
        <div class="ad-count">{{ countdown }}s</div>
      </div>
      <p class="ad-note">Reward unlocks after the ad finishes</p>
      <button class="ad-cancel" @click="emit('cancel')">✕</button>
    </div>
  </div>
</template>

<style scoped>
.ad-overlay {
  position: fixed; inset: 0; z-index: 130;
  background: rgba(0, 0, 0, .85);
  display: flex; align-items: center; justify-content: center;
}
.ad-box { width: 82%; max-width: 340px; position: relative; }
.ad-video {
  position: relative;
  aspect-ratio: 4/3;
  border-radius: 14px;
  background: linear-gradient(135deg, #1f2937, #111827);
  display: flex; align-items: center; justify-content: center;
}
.ad-placeholder { color: #9ca3af; font-size: 15px; }
.ad-count {
  position: absolute; top: 10px; right: 12px;
  background: rgba(0,0,0,.6); border-radius: 8px;
  padding: 3px 9px; font-size: 12px; font-weight: 700;
}
.ad-note { text-align: center; color: #9a9aa5; font-size: 12px; margin-top: 10px; }
.ad-cancel {
  position: absolute; top: -38px; right: 0;
  width: 28px; height: 28px; border-radius: 50%;
  background: rgba(255,255,255,.15); color: #fff; font-size: 13px;
}
</style>
