<script setup lang="ts">
import ATag from 'ant-design-vue/es/tag'

defineProps<{
  label: string
  color?: string
}>()
</script>

<template>
  <a-tag class="status-tag" :color="color">
    {{ label }}
  </a-tag>
</template>
