<script setup lang="ts">
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import AResult from 'ant-design-vue/es/result'
import { useMessages } from '../i18n'

const router = useRouter()
const { t } = useMessages({
  en: {
    title: 'Page not found',
    subtitle: 'This route is not part of the task-oriented AgentMeter workspace.',
    action: 'Go to Attention'
  },
  'zh-CN': {
    title: '页面不存在',
    subtitle: '此路径不属于新版任务导向的 AgentMeter 工作区。',
    action: '前往关注'
  }
})
</script>

<template>
  <div class="page">
    <a-result status="404" :title="t('title')" :sub-title="t('subtitle')">
      <template #extra>
        <a-button type="primary" @click="router.push('/attention')">{{ t('action') }}</a-button>
      </template>
    </a-result>
  </div>
</template>
