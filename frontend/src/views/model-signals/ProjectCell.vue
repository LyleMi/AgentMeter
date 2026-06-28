<script setup lang="ts">
import ATooltip from 'ant-design-vue/es/tooltip'

defineProps<{
  info: {
    main: string
    full?: string
  }
  showMeta?: boolean
}>()
</script>

<template>
  <div class="model-signals-project-cell">
    <a-tooltip :title="info.full" placement="topLeft">
      <span class="model-signals-project">{{ info.main }}</span>
    </a-tooltip>
    <div v-if="showMeta" class="source-identity-meta">{{ info.full }}</div>
  </div>
</template>

<style scoped>
.model-signals-project-cell {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.model-signals-project {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: var(--am-text);
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.source-identity-meta {
  overflow: hidden;
  color: var(--am-muted);
  font-size: 12px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
