<script setup lang="ts">
import type { Component } from 'vue'
import Panel from '../../components/ui/Panel.vue'

defineProps<{
  title: string
  kicker?: string
  icon?: Component
}>()
</script>

<template>
  <Panel class="model-signals-table-panel" :title="title" :kicker="kicker" :icon="icon">
    <slot />
  </Panel>
</template>

<style scoped>
.model-signals-table-panel :deep(.panel-body) {
  padding: 0;
}

.model-signals-table-panel :deep(.model-signals-warning-row td) {
  background: rgba(254, 243, 199, 0.36);
}

.model-signals-table-panel :deep(.model-signals-critical-row td) {
  background: rgba(254, 226, 226, 0.46);
}
</style>
