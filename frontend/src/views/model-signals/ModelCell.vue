<script setup lang="ts">
defineProps<{
  model?: string
  provider?: string
  fallback: string
  showProvider?: boolean
}>()
</script>

<template>
  <div class="model-signals-model-cell">
    <span class="model-name">{{ model || fallback }}</span>
    <div v-if="showProvider" class="source-identity-meta">{{ provider || '-' }}</div>
  </div>
</template>

<style scoped>
.model-signals-model-cell {
  display: grid;
  gap: 2px;
  min-width: 0;
}
</style>
