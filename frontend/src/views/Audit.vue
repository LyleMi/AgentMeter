<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import ASelect from 'ant-design-vue/es/select'
import {
  BarChartOutlined,
  FileSearchOutlined,
  UnorderedListOutlined
} from '@ant-design/icons-vue'
import { api } from '../api/client'
import PageHeader from '../components/PageHeader.vue'
import PageTabs from '../components/PageTabs.vue'
import { useMessages } from '../i18n'
import { auditPath, cleanQueryValue, cleanRouteQuery } from './auditSupport'

const route = useRoute()
const router = useRouter()
const agentOptions = ref<Array<{ value: string; label: string }>>([])
const agentLoading = ref(false)
const { t } = useMessages({
  en: {
    'title': 'Audit',
    'subtitle': 'Command and privacy findings split by summary, finding list, and session-linked detail',
    'filter.agent': 'Agent',
    'tab.summary': 'Summary',
    'tab.list': 'Findings',
    'tab.detail': 'Detail'
  },
  'zh-CN': {
    'title': '审计',
    'subtitle': '按汇总、发现列表和会话关联详情拆分命令与隐私发现',
    'filter.agent': 'Agent',
    'tab.summary': '汇总',
    'tab.list': '发现',
    'tab.detail': '详情'
  }
})

const selectedAgent = computed(() => cleanQueryValue(route.query.agent) || undefined)
const visibleAgentOptions = computed(() => {
  if (!selectedAgent.value || agentOptions.value.some((item) => item.value === selectedAgent.value)) {
    return agentOptions.value
  }
  return [{ value: selectedAgent.value, label: selectedAgent.value }, ...agentOptions.value]
})

const activeKey = computed(() => {
  if (route.path.startsWith('/audit/findings/')) return 'detail'
  if (route.path.startsWith('/audit/findings')) return 'list'
  return 'summary'
})

const tabs = computed(() => {
  const items = [
    { key: 'summary', label: t('tab.summary'), path: auditPath('/audit/summary', { agent: selectedAgent.value }), icon: BarChartOutlined },
    { key: 'list', label: t('tab.list'), path: auditPath('/audit/findings', { agent: selectedAgent.value }), icon: UnorderedListOutlined }
  ]
  if (activeKey.value === 'detail') {
    items.push({ key: 'detail', label: t('tab.detail'), path: route.fullPath, icon: FileSearchOutlined })
  }
  return items
})

async function loadAgents() {
  agentLoading.value = true
  try {
    const overview = await api.getOverview()
    const options = new Map<string, string>()
    const usage = overview.agentUsage || []
    usage.forEach((item) => {
      if (item.agentKind) options.set(item.agentKind, item.agentName || item.agentKind)
    })
    agentOptions.value = [...options.entries()]
      .sort((left, right) => left[1].localeCompare(right[1]))
      .map(([value, label]) => ({ value, label }))
  } finally {
    agentLoading.value = false
  }
}

function updateAgent(value?: unknown) {
  const agent = cleanQueryValue(value) || undefined
  const query = cleanRouteQuery(route.query)
  if (agent) {
    query.agent = agent
  } else {
    delete query.agent
  }
  router.push({ path: route.path, query })
}

onMounted(loadAgents)
</script>

<template>
  <div class="page audit-page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')">
      <template #actions>
        <a-select
          class="audit-agent-filter"
          allow-clear
          :loading="agentLoading"
          :placeholder="t('filter.agent')"
          :options="visibleAgentOptions"
          :value="selectedAgent"
          @change="updateAgent"
        />
      </template>
    </PageHeader>

    <PageTabs class="audit-subnav" :tabs="tabs" :active-key="activeKey" />

    <RouterView />
  </div>
</template>

<style scoped>
.audit-page {
  max-width: 1560px;
}

.audit-agent-filter {
  width: 220px;
  max-width: 100%;
}
</style>
