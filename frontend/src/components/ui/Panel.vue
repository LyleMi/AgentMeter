<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  title: string
  kicker?: string
  icon?: Component
}>()
</script>

<template>
  <section class="panel">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ title }}</h2>
        <div v-if="kicker" class="panel-kicker">{{ kicker }}</div>
      </div>
      <div v-if="icon || $slots.actions" class="panel-actions">
        <component :is="icon" v-if="icon" class="panel-header-icon" />
        <slot name="actions" />
      </div>
    </div>
    <div class="panel-body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="panel-footer-note">
      <slot name="footer" />
    </div>
  </section>
</template>
