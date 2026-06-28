<script setup lang="ts">
defineProps<{
  info: {
    label: string
    secondary?: string
  }
}>()
</script>

<template>
  <div class="model-signals-source-cell">
    <span class="source-identity-name">{{ info.label }}</span>
    <div class="source-identity-meta">{{ info.secondary || '-' }}</div>
  </div>
</template>

<style scoped>
.model-signals-source-cell {
  display: grid;
  gap: 2px;
  min-width: 0;
}
</style>
