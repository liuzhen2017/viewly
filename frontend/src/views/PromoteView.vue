<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { store, refreshMe } from '../store'

// Promote flow: TKPostAI has no server-side register API, so the one-time
// account binding happens on their page via the publish link. Once the user
// confirms binding (tk_synced), every promote tap opens their publish
// window directly — their platform auto-logs-in bound users.
const dramas = ref([])
const page = ref(1)
const loading = ref(false)
const busy = ref(0)
const openedFirst = ref(false) // a link was opened this visit (pre-bind step)

const synced = computed(() => !!store.user?.tk_synced)

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

async function openLink(drama) {
  busy.value = drama.id
  try {
    const r = await api.tkLink(drama.id)
    if (!r.url) throw new Error('no link returned')
    window.open(r.url, '_blank') // new tab so the user returns here to confirm
    openedFirst.value = true
    setTimeout(() => { busy.value = 0 }, 1500)
  } catch (e) {
    window.$toast(e.message)
    busy.value = 0
  }
}

async function confirmSynced() {
  try {
    await api.tkSynced()
    await refreshMe()
    window.$toast('Promote access ready ✓')
  } catch (e) { window.$toast(e.message) }
}
</script>

<template>
  <div class="page">
    <div class="head">
      <h2>🚀 Promote &amp; Earn</h2>
      <p class="tip">Pick a drama, post promo videos to your TikTok/Facebook,
        and get credited for the traffic you bring.</p>
    </div>

    <div v-if="!store.user" class="card hint-card">
      <router-link to="/login" class="big-link">Log in first to promote</router-link>
    </div>

    <!-- step 1: one-time account binding on the promote platform -->
    <div v-else-if="!synced" class="card hint-card">
      <div class="step-title">① First time — connect your promote account</div>
      <p class="step-text">
        Tap any drama below to open the promote platform. Create your account
        there <b>(one time, ~30s)</b> — after that, promoting goes straight to
        the publish page, no registration again.
      </p>
      <div v-if="openedFirst" class="confirm-row">
        <span class="ask">Done binding? Unlock one-tap promoting:</span>
        <button class="confirm-btn" @click="confirmSynced">✓ I've finished binding</button>
      </div>
    </div>
    <div v-else class="card ok-card">✓ Promote account connected — tap a drama to publish instantly</div>

    <div class="grid">
      <div v-for="d in dramas" :key="d.id" class="cell" @click="openLink(d)">
        <div class="poster">
          <img :src="d.cover" alt="" loading="lazy" />
          <span class="eps">{{ d.episodes }} EP</span>
        </div>
        <div class="name">{{ d.title }}</div>
        <button class="go" :disabled="busy === d.id">
          {{ busy === d.id ? 'Opening…' : (synced ? '🚀 Promote' : '① Connect & Promote') }}
        </button>
      </div>
    </div>

    <button v-if="dramas.length && dramas.length % 24 === 0" class="more" @click="loadMore">Load more</button>
    <div v-if="!dramas.length && !loading" class="empty">No dramas yet</div>
  </div>
</template>

<style scoped>
.page { padding: 14px 14px calc(var(--nav-h) + 20px); }
.head h2 { font-size: 19px; margin-bottom: 6px; }
.tip { color: var(--muted); font-size: 12.5px; line-height: 1.6; margin-bottom: 14px; }
.hint-card, .ok-card { padding: 14px; margin-bottom: 14px; font-size: 13px; }
.ok-card { color: #8fe3a2; }
.big-link { color: var(--accent); font-weight: 700; font-size: 15px; }
.step-title { font-weight: 800; margin-bottom: 6px; }
.step-text { color: var(--muted); line-height: 1.6; margin: 0; }
.confirm-row { margin-top: 12px; display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.ask { font-size: 12.5px; }
.confirm-btn {
  border: none; cursor: pointer; padding: 9px 16px; border-radius: 8px;
  background: #2fae62; color: #fff; font-weight: 700; font-size: 13px;
}
.grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
.cell { cursor: pointer; }
.poster { position: relative; border-radius: 9px; overflow: hidden; aspect-ratio: 3/4; background: var(--bg-elev2); }
.poster img { width: 100%; height: 100%; object-fit: cover; display: block; }
.eps { position: absolute; right: 5px; bottom: 5px; background: rgba(0,0,0,.62); color: #fff; font-size: 10px; padding: 2px 6px; border-radius: 4px; }
.name { font-size: 12px; margin: 6px 0 5px; line-height: 1.35; height: 2.7em; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
.go { width: 100%; border: none; cursor: pointer; background: linear-gradient(120deg, #ff5a3c, #ff8a4c); color: #fff; font-size: 11.5px; font-weight: 700; padding: 7px 0; border-radius: 7px; }
.go:disabled { opacity: .55; }
.more { display: block; margin: 16px auto 0; padding: 10px 26px; background: var(--bg-elev2); color: var(--muted); border: none; border-radius: 8px; }
.empty { text-align: center; color: var(--muted); padding: 60px 0; }
</style>
