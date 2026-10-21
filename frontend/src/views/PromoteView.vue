<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { store } from '../store'

// 我要推广: pick a drama -> our server signs with TKPostAI -> open the
// publish link in this tab; the user binds social accounts and posts videos
// there. Tasks are attributed to their user ID + this drama.
const dramas = ref([])
const page = ref(1)
const loading = ref(false)
const busy = ref(0)

onMounted(loadMore)

async function loadMore() {
  if (loading.value) return
  loading.value = true
  try {
    const r = await api.dramas({ page: page.value, size: 24 })
    dramas.value.push(...(r.list || []))
    page.value++
  } catch (e) { window.$toast(e.message) }
  loading.value = false
}

async function promote(d) {
  busy.value = d.id
  try {
    const r = await api.tkLink(d.id)
    if (!r.url) throw new Error('no link returned')
    location.href = r.url // same tab, per TKPostAI's guidance
  } catch (e) {
    window.$toast(e.message)
    busy.value = 0
  }
}
</script>

<template>
  <div class="page">
    <div class="head">
      <h2>🚀 Promote &amp; Earn</h2>
      <p class="tip">
        Pick a drama → get your publish link → connect TikTok/Facebook and post
        promo videos. Tasks are tracked under your account.
      </p>
    </div>

    <div v-if="!store.user" class="card login-hint">
      <router-link to="/login" class="login-btn">Log in first to promote</router-link>
    </div>

    <div class="grid">
      <div v-for="d in dramas" :key="d.id" class="cell" @click="promote(d)">
        <div class="poster">
          <img :src="d.cover" alt="" loading="lazy" />
          <span class="eps">{{ d.episodes }} EP</span>
        </div>
        <div class="name">{{ d.title }}</div>
        <button class="go" :disabled="busy === d.id">
          {{ busy === d.id ? 'Getting link…' : '🚀 Promote this' }}
        </button>
      </div>
    </div>

    <button v-if="dramas.length && dramas.length % 24 === 0" class="more" @click="loadMore">
      Load more
    </button>
    <div v-if="!dramas.length && !loading" class="empty">No dramas yet</div>
  </div>
</template>

<style scoped>
.page { padding: 14px 14px calc(var(--nav-h) + 20px); }
.head h2 { font-size: 19px; margin-bottom: 6px; }
.tip { color: var(--muted); font-size: 12.5px; line-height: 1.6; margin-bottom: 14px; }
.login-hint { text-align: center; padding: 18px; margin-bottom: 14px; }
.login-btn { color: var(--accent); font-weight: 700; font-size: 15px; }
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
.cell { cursor: pointer; }
.poster { position: relative; border-radius: 9px; overflow: hidden; aspect-ratio: 3/4; background: var(--bg-elev2); }
.poster img { width: 100%; height: 100%; object-fit: cover; display: block; }
.eps {
  position: absolute; right: 5px; bottom: 5px;
  background: rgba(0,0,0,.62); color: #fff;
  font-size: 10px; padding: 2px 6px; border-radius: 4px;
}
.name {
  font-size: 12px; margin: 6px 0 5px; line-height: 1.35; height: 2.7em;
  overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
}
.go {
  width: 100%; border: none; cursor: pointer;
  background: linear-gradient(120deg, #ff5a3c, #ff8a4c); color: #fff;
  font-size: 11.5px; font-weight: 700; padding: 7px 0; border-radius: 7px;
}
.go:disabled { opacity: .55; }
.more {
  display: block; margin: 16px auto 0; padding: 10px 26px;
  background: var(--bg-elev2); color: var(--muted); border: none; border-radius: 8px;
}
.empty { text-align: center; color: var(--muted); padding: 60px 0; }
</style>
