<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { admin, tenantSlug } from '../api'
import { t } from '../i18n'

const s = ref(null)
const busy = ref(false)

onMounted(async () => {
  s.value = await admin.adSettings()
})

async function save() {
  busy.value = true
  try {
    s.value = await admin.saveAdSettings({
      adsense_client: s.value.adsense_client,
      adsense_enabled: s.value.adsense_enabled,
      rewarded_ad_mode: s.value.rewarded_ad_mode,
      rewarded_ad_coins: s.value.rewarded_ad_coins,
      rewarded_ad_daily_limit: s.value.rewarded_ad_daily_limit,
      admob_app_id: s.value.admob_app_id,
      admob_rewarded_unit_id: s.value.admob_rewarded_unit_id,
    })
    ElMessage.success(t('saved'))
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div v-if="s" style="display:flex;gap:14px;align-items:flex-start;flex-wrap:wrap">
    <el-card style="flex:1;min-width:380px">
      <template #header>📰 AdSense（H5 {{ t('adWeb') }}）</template>
      <el-form label-width="150px">
        <el-form-item :label="t('adEnabled')">
          <el-switch v-model="s.adsense_enabled" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="Publisher ID">
          <el-input v-model="s.adsense_client" placeholder="ca-pub-0000000000000000" />
        </el-form-item>
      </el-form>
      <el-alert type="info" :closable="false" style="margin-top:6px">
        <p style="margin:0;font-size:12px;line-height:1.8">
          1. {{ tenantSlug() }}.yourdomain.com/ads.txt → <code>google.com, {pub-id}, DIRECT, f08c47fec0942fa0</code> {{ t('adAutoServed') }}<br />
          2. {{ t('adReviewNote') }} /privacy /terms {{ t('adReviewNote2') }}
        </p>
      </el-alert>
    </el-card>

    <el-card style="flex:1;min-width:380px">
      <template #header>📺 {{ t('adRewardedTitle') }}</template>
      <el-form label-width="150px">
        <el-form-item :label="t('adMode')">
          <el-radio-group v-model="s.rewarded_ad_mode">
            <el-radio-button value="off">{{ t('adModeOff') }}</el-radio-button>
            <el-radio-button value="client">H5</el-radio-button>
            <el-radio-button value="ssv">App SSV</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('adCoinsPer')">
          <el-input-number v-model="s.rewarded_ad_coins" :min="0" :max="10000" />
        </el-form-item>
        <el-form-item :label="t('adDailyLimit')">
          <el-input-number v-model="s.rewarded_ad_daily_limit" :min="0" :max="100" />
        </el-form-item>
      </el-form>
      <el-divider>AdMob (App SSV)</el-divider>
      <el-form label-width="150px">
        <el-form-item label="AdMob App ID">
          <el-input v-model="s.admob_app_id" placeholder="ca-app-pub-…~1234567890" />
        </el-form-item>
        <el-form-item :label="t('adRewardedUnit')">
          <el-input v-model="s.admob_rewarded_unit_id" placeholder="ca-app-pub-…/1234567890" />
        </el-form-item>
      </el-form>
      <el-alert type="warning" :closable="false">
        <p style="margin:0;font-size:12px;line-height:1.8">
          SSV callback URL: <code>https://api.yourdomain.com/api/webhooks/admob/ssv</code><br />
          custom_data: <code>uid=&lt;user_id&gt;&amp;tenant={{ tenantSlug() }}</code><br />
          {{ t('adSsvNote') }}
        </p>
      </el-alert>
    </el-card>

    <div style="width:100%">
      <el-button type="primary" :loading="busy" @click="save">{{ t('save') }}</el-button>
    </div>
  </div>
  <div v-else class="el-loading-mask" style="height:200px"></div>
</template>
