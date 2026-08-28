<script setup>
import { useRoute } from 'vue-router'
const route = useRoute()
const isPrivacy = route.path === '/privacy'
const brand = 'FoxDrama'
const year = new Date().getFullYear()
</script>

<template>
  <div class="page legal">
    <button class="back" @click="$router.back()">‹</button>
    <h1>{{ isPrivacy ? 'Privacy Policy' : 'Terms of Service' }}</h1>
    <p class="updated">Last updated: {{ year }}</p>

    <template v-if="isPrivacy">
      <h2>1. Information We Collect</h2>
      <p>When you use {{ brand }}, we collect: (a) device identifiers used to create and restore your guest account; (b) your email address, only if you choose to register; (c) watching activity (dramas, episodes, progress) to provide continue-watching; (d) transaction records for coin purchases.</p>
      <h2>2. How We Use Information</h2>
      <p>To operate the service: maintaining your coin balance and unlocks, processing purchases, personalizing recommendations, and preventing fraud and abuse of the rewards system.</p>
      <h2>3. Third-Party Services</h2>
      <p>We use third-party providers for payments and advertising. These providers may process data under their own policies when you interact with them (for example, Google payments or Google Ads, including personalized advertising where permitted). You can opt out of personalized ads in your device or browser settings.</p>
      <h2>4. Data Retention & Your Rights</h2>
      <p>Account data is retained while your account is active. You may request deletion of your account and personal data by contacting support. Deleting your account forfeits remaining coins and purchases.</p>
      <h2>5. Children</h2>
      <p>{{ brand }} is not directed to children under 13, and we do not knowingly collect their personal data.</p>
      <h2>6. Contact</h2>
      <p>Questions about this policy can be sent through the in-app contact channel.</p>
    </template>

    <template v-else>
      <h2>1. Acceptance</h2>
      <p>By accessing {{ brand }} you agree to these terms. If you do not agree, please do not use the service.</p>
      <h2>2. Virtual Coins</h2>
      <p>Coins are a virtual license to access paid episodes. They are non-transferable, have no cash value, and are not refundable except where required by law. Earned coins from check-ins, tasks, and rewarded ads follow the daily limits displayed in the Rewards section.</p>
      <h2>3. VIP Subscription</h2>
      <p>VIP grants unlimited episode access during the subscription period. Subscriptions renew automatically unless cancelled before the renewal date. You may cancel at any time; access continues until the period ends.</p>
      <h2>4. Content Rules</h2>
      <p>All dramas, artwork, and videos on {{ brand }} are licensed to the operator of this site. You may not download, redistribute, or rebroadcast content. Automated access, scraping, or abuse of the rewards system may result in account suspension and forfeiture of coins.</p>
      <h2>5. Changes</h2>
      <p>We may update these terms; material changes will be announced in-app. Continued use after changes constitutes acceptance.</p>
    </template>
  </div>
</template>

<style scoped>
.legal { line-height: 1.7; }
.back { font-size: 26px; color: var(--muted); }
h1 { font-size: 22px; margin: 8px 0 2px; }
.updated { color: var(--muted); font-size: 12px; margin-bottom: 20px; }
h2 { font-size: 15px; margin: 20px 0 6px; }
p { color: #c8c8d0; font-size: 13.5px; margin: 0; }
</style>
