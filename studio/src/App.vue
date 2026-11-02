<script setup>
import { ElConfigProvider } from 'element-plus'
import { epLocale } from './i18n'
</script>

<template>
  <el-config-provider :locale="epLocale">
    <router-view />
  </el-config-provider>
</template>
