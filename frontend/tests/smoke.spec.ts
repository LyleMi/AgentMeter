import { expect, test, type Page } from '@playwright/test'

interface SmokeRoute {
  path: string
  hash: string
  title: RegExp
  panelTitle?: RegExp
}

const routes: SmokeRoute[] = [
  { path: '/#/attention', hash: '#/attention', title: /^(Attention|关注)$/, panelTitle: /^(Needs attention|需要关注)$/ },
  { path: '/#/analysis/usage', hash: '#/analysis/usage', title: /^(Usage|用量)$/, panelTitle: /^(Token Mix|Token 构成)$/ },
  { path: '/#/analysis/usage/projects', hash: '#/analysis/usage/projects', title: /^(Usage|用量)$/, panelTitle: /^(Project spend|项目消耗)$/ },
  { path: '/#/analysis/usage/breakdown', hash: '#/analysis/usage/breakdown', title: /^(Usage|用量)$/, panelTitle: /^(Usage Breakdown|用量拆分)$/ },
  { path: '/#/analysis/usage/sessions', hash: '#/analysis/usage/sessions', title: /^(Usage|用量)$/, panelTitle: /^(High Token Sessions|高 Token 会话)$/ },
  { path: '/#/analysis/time', hash: '#/analysis/time', title: /^(Time|耗时)$/ },
  { path: '/#/analysis/time/sources', hash: '#/analysis/time/sources', title: /^(Time|耗时)$/ },
  { path: '/#/analysis/time/tools', hash: '#/analysis/time/tools', title: /^(Time|耗时)$/ },
  { path: '/#/analysis/time/sessions', hash: '#/analysis/time/sessions', title: /^(Time|耗时)$/ },
  { path: '/#/analysis/models', hash: '#/analysis/models', title: /^(Model Signals|模型表现)$/ },
  { path: '/#/analysis/models/trends', hash: '#/analysis/models/trends', title: /^(Model Signals|模型表现)$/ },
  { path: '/#/analysis/models/compare', hash: '#/analysis/models/compare', title: /^(Model Signals|模型表现)$/ },
  { path: '/#/analysis/tools', hash: '#/analysis/tools', title: /^(Tools|工具)$/ },
  { path: '/#/analysis/tools/shell', hash: '#/analysis/tools/shell', title: /^(Tools|工具)$/ },
  { path: '/#/analysis/tools/calls', hash: '#/analysis/tools/calls', title: /^(Tools|工具)$/ },
  { path: '/#/sessions', hash: '#/sessions', title: /^(Sessions|会话)$/ },
  { path: '/#/safety/audit', hash: '#/safety/audit', title: /^(Audit|审计)$/ },
  { path: '/#/safety/audit/findings', hash: '#/safety/audit/findings', title: /^(Audit|审计)$/ },
  { path: '/#/safety/privacy', hash: '#/safety/privacy', title: /^(Agent Privacy|Agent 隐私)$/ },
  { path: '/#/resources/prompts', hash: '#/resources/prompts', title: /^(Prompts|Prompt)$/ },
  { path: '/#/resources/agents', hash: '#/resources/agents', title: /^(Agent Resources|Agent 资源)$/ },
  { path: '/#/settings/sources', hash: '#/settings/sources', title: /^(Settings|设置)$/ },
  { path: '/#/settings/data', hash: '#/settings/data', title: /^(Settings|设置)$/ },
  { path: '/#/settings/pricing', hash: '#/settings/pricing', title: /^(Settings|设置)$/ },
  { path: '/#/settings/display', hash: '#/settings/display', title: /^(Settings|设置)$/ }
]

const ignoredConsoleErrorPatterns = [
  /Warning: \[ant-design-vue: Typography\] When `ellipsis` is enabled, please use `content` instead of children/
]

test('task-oriented hash routes render without API or console failures', async ({ page }) => {
  test.setTimeout(120_000)
  const failures = watchPageFailures(page)

  for (const route of routes) {
    await test.step(route.hash, async () => {
      await page.goto(route.path, { waitUntil: 'domcontentloaded' })
      await expect(page).toHaveURL(new RegExp(`${escapeRegExp(route.hash)}$`))
      await expect(page.locator('.app-shell')).toBeVisible()
      await expect(page.locator('h1.page-title, h2.panel-title').filter({ hasText: route.title }).first()).toBeVisible()
      if (route.panelTitle) {
        await expect(page.locator('h2.panel-title').filter({ hasText: route.panelTitle }).first()).toBeVisible()
      }
      await settleRoute(page)
    })
  }

  expect(failures.apiResponses, formatFailures('API 5xx responses', failures.apiResponses)).toHaveLength(0)
  expect(failures.consoleErrors, formatFailures('Console/page errors', failures.consoleErrors)).toHaveLength(0)
})

test('filters persist across analysis tasks and attention drills down', async ({ page }) => {
  await page.goto('/#/attention?range=week&model=gpt-5-codex')
  await expect(page.locator('.attention-list-panel')).toBeVisible()
  const review = page.locator('.attention-item .ant-btn').first()
  if (await review.isVisible()) {
    await review.click()
    await expect(page).toHaveURL(/#\/(sessions\/\d+|safety\/audit\/findings\/\d+|analysis\/models|settings\/|safety\/privacy)/)
  }

  await page.goto('/#/analysis/usage?range=week&model=gpt-5-codex')
  await page.locator('.task-section-nav .ant-btn').filter({ hasText: /^(Time|耗时)$/ }).click()
  await expect(page).toHaveURL(/#\/analysis\/time\?/)
  expect(new URL(page.url()).hash).toContain('model=gpt-5-codex')
  expect(new URL(page.url()).hash).toContain('range=week')
})

test('old routes show not found and mobile pages do not overflow', async ({ page }) => {
  for (const oldPath of ['/#/overview', '/#/tokens', '/#/time', '/#/model-signals', '/#/tools', '/#/audit', '/#/agent-privacy']) {
    await page.goto(oldPath)
    await expect(page.locator('.ant-result-title')).toContainText(/Page not found|页面不存在/)
  }

  await page.setViewportSize({ width: 390, height: 844 })
  for (const path of ['/#/attention', '/#/analysis/usage', '/#/sessions', '/#/safety/audit']) {
    await page.goto(path)
    const overflows = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)
    expect(overflows, `${path} should not overflow horizontally`).toBe(false)
  }
})

function watchPageFailures(page: Page) {
  const consoleErrors: string[] = []
  const apiResponses: string[] = []
  page.on('console', (message) => {
    if (message.type() !== 'error') return
    if (ignoredConsoleErrorPatterns.some((pattern) => pattern.test(message.text()))) return
    consoleErrors.push(message.text())
  })
  page.on('pageerror', (error) => consoleErrors.push(error.stack || error.message))
  page.on('response', (response) => {
    if (new URL(response.url()).pathname.startsWith('/api') && response.status() >= 500) {
      apiResponses.push(`${response.status()} ${response.request().method()} ${response.url()}`)
    }
  })
  return { consoleErrors, apiResponses }
}

async function settleRoute(page: Page) {
  await page.waitForLoadState('networkidle', { timeout: 750 }).catch(() => undefined)
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function formatFailures(label: string, failures: string[]) {
  return failures.length ? `${label}:\n${[...new Set(failures)].join('\n')}` : label
}
