<script setup lang="ts">
import ATag from 'ant-design-vue/es/tag'

defineProps<{
  reasons?: readonly string[]
  color?: string
  emptyText: string
}>()
</script>

<template>
  <div v-if="reasons?.length" class="model-signals-tags">
    <a-tag v-for="reason in reasons" :key="reason" :color="color">
      {{ reason }}
    </a-tag>
  </div>
  <span v-else class="muted">{{ emptyText }}</span>
</template>

<style scoped>
.model-signals-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.model-signals-tags .ant-tag {
  max-width: 100%;
  margin-right: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
