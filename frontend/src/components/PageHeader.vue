<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
}>()
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">{{ title }}</h1>
      <div v-if="subtitle" class="page-subtitle">{{ subtitle }}</div>
    </div>
    <slot name="actions" />
  </div>
</template>
