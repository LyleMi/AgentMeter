<script setup lang="ts">
defineProps<{
  compact?: boolean
}>()
</script>

<template>
  <div :class="['toolbar', { 'toolbar-compact': compact }]">
    <div v-if="$slots.left" class="toolbar-left">
      <slot name="left" />
    </div>
    <div v-if="$slots.right" class="toolbar-right">
      <slot name="right" />
    </div>
  </div>
</template>
