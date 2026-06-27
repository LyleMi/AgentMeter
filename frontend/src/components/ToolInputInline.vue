<script setup lang="ts">
import { computed } from 'vue'
import ATooltip from 'ant-design-vue/es/tooltip'
import type { ToolCall } from '../api'
import { parseToolInput } from '../toolInput'

const props = defineProps<{
  call: ToolCall
}>()

const parsed = computed(() => parseToolInput(props.call))
const visibleFields = computed(() => parsed.value.fields.slice(0, 3))
const hiddenFieldCount = computed(() => Math.max(parsed.value.fields.length - visibleFields.value.length, 0))
</script>

<template>
  <a-tooltip :title="parsed.tooltip || '-'" placement="topLeft">
    <div class="tool-input-inline" :class="{ 'is-empty': !parsed.hasInput }">
      <template v-if="visibleFields.length">
        <span v-for="field in visibleFields" :key="field.key" class="tool-input-token">
          <span class="tool-input-token-key">{{ field.key }}</span>
          <span class="tool-input-token-value">{{ field.preview || '-' }}</span>
        </span>
        <span v-if="hiddenFieldCount" class="tool-input-more">+{{ hiddenFieldCount }}</span>
      </template>
      <span v-else class="tool-input-fallback">{{ parsed.preview || '-' }}</span>
    </div>
  </a-tooltip>
</template>
