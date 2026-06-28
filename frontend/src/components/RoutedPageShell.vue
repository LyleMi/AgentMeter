<script setup lang="ts">
import { RouterView, useRouter } from 'vue-router'
import PageHeader from './PageHeader.vue'
import PageTabs from './PageTabs.vue'
import type { PageTabItem } from './PageTabs.vue'

defineProps<{
  title: string
  subtitle?: string
  tabs: PageTabItem[]
  activeKey: string
}>()

const router = useRouter()

function selectTab(item: PageTabItem) {
  router.push(item.path)
}
</script>

<template>
  <div class="page">
    <PageHeader :title="title" :subtitle="subtitle" />
    <PageTabs :tabs="tabs" :active-key="activeKey" @select="selectTab" />
    <RouterView />
  </div>
</template>
