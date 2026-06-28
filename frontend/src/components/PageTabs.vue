<script setup lang="ts">
import type { Component } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'

interface PageTabItem {
  key: string
  label: string
  path: string
  icon: Component
}

defineProps<{
  tabs: PageTabItem[]
  activeKey: string
}>()

const router = useRouter()

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <div class="page-tabs">
    <a-button
      v-for="item in tabs"
      :key="item.key"
      :type="item.key === activeKey ? 'primary' : 'default'"
      @click="navigate(item.path)"
    >
      <template #icon>
        <component :is="item.icon" />
      </template>
      {{ item.label }}
    </a-button>
  </div>
</template>
