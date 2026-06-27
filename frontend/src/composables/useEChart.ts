import { onBeforeUnmount, onMounted, ref } from 'vue'
import { init, type ECharts } from '../chartRuntime'

export function useEChart() {
  const chartEl = ref<HTMLDivElement | null>(null)
  let chart: ECharts | null = null

  function getChart() {
    if (!chartEl.value) return null
    if (!chart) chart = init(chartEl.value)
    return chart
  }

  function disposeChart() {
    chart?.dispose()
    chart = null
  }

  function resizeChart() {
    chart?.resize()
  }

  onMounted(() => {
    window.addEventListener('resize', resizeChart)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('resize', resizeChart)
    disposeChart()
  })

  return {
    chartEl,
    getChart,
    disposeChart,
    resizeChart
  }
}
