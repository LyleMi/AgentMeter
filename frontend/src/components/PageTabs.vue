<script setup lang="ts">
import { getCurrentInstance } from 'vue'
import type { Component } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'

export interface PageTabItem {
  key: string
  label: string
  path: string
  icon: Component
}

defineProps<{
  tabs: PageTabItem[]
  activeKey: string
}>()

const emit = defineEmits<{
  select: [item: PageTabItem]
}>()

const instance = getCurrentInstance()
const router = useRouter()

function hasSelectListener() {
  return Boolean(instance?.vnode.props?.onSelect)
}

function selectTab(item: PageTabItem) {
  emit('select', item)

  if (!hasSelectListener()) {
    router.push(item.path)
  }
}
</script>

<template>
  <div class="page-tabs">
    <a-button
      v-for="item in tabs"
      :key="item.key"
      :type="item.key === activeKey ? 'primary' : 'default'"
      @click="selectTab(item)"
    >
      <template #icon>
        <component :is="item.icon" />
      </template>
      {{ item.label }}
    </a-button>
  </div>
</template>
