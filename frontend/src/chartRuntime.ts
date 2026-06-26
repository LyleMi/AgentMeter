import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { init, use, type ECharts } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'

use([BarChart, LineChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer])

export { init, type ECharts }
