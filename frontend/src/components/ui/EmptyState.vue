<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  title: string
  text?: string
  compact?: boolean
  icon?: Component
}>()
</script>

<template>
  <div :class="['empty-state', { 'empty-state-compact': compact }]">
    <component :is="icon" v-if="icon" class="empty-state-icon" />
    <div v-else-if="$slots.icon" class="empty-state-icon">
      <slot name="icon" />
    </div>
    <div class="empty-state-title">{{ title }}</div>
    <div v-if="text" class="empty-state-text">{{ text }}</div>
  </div>
</template>
