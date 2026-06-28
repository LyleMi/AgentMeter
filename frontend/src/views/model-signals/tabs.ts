import type { Component } from 'vue'
import {
  BranchesOutlined,
  CalendarOutlined,
  DashboardOutlined,
  LineChartOutlined,
  TableOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import type { ModelSignalsTranslate } from './messages'

export type ModelSignalsTabKey = 'charts' | 'overview' | 'daily' | 'cohorts' | 'matrix' | 'projects' | 'anomalies'

export interface ModelSignalsTab {
  key: ModelSignalsTabKey
  label: string
  icon: Component
}

export function buildModelSignalsTabs(t: ModelSignalsTranslate): ModelSignalsTab[] {
  return [
    { key: 'charts', label: t('tab.charts'), icon: LineChartOutlined },
    { key: 'overview', label: t('tab.overview'), icon: DashboardOutlined },
    { key: 'daily', label: t('tab.daily'), icon: CalendarOutlined },
    { key: 'cohorts', label: t('tab.cohorts'), icon: BranchesOutlined },
    { key: 'matrix', label: t('tab.matrix'), icon: TableOutlined },
    { key: 'projects', label: t('tab.projects'), icon: LineChartOutlined },
    { key: 'anomalies', label: t('tab.anomalies'), icon: WarningOutlined }
  ]
}
