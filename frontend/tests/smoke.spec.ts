import { expect, test, type Page } from '@playwright/test'

interface SmokeRoute {
  path: string
  hash: string
  title: RegExp
  panelTitle?: RegExp
}

const routes: SmokeRoute[] = [
  { path: '/#/overview/summary', hash: '#/overview/summary', title: /^(Overview|概览)$/ },
  { path: '/#/time', hash: '#/time', title: /^(Time|耗时)$/ },
  { path: '/#/sessions', hash: '#/sessions', title: /^(Sessions|会话)$/ },
  { path: '/#/tokens', hash: '#/tokens', title: /^(Tokens|Token)$/, panelTitle: /^(Token Mix|Token 构成)$/ },
  { path: '/#/tokens/trends', hash: '#/tokens/trends', title: /^(Tokens|Token)$/, panelTitle: /^(Cache Hit Trend|缓存命中趋势)$/ },
  { path: '/#/tokens/breakdown', hash: '#/tokens/breakdown', title: /^(Tokens|Token)$/, panelTitle: /^(Usage Breakdown|用量拆分)$/ },
  { path: '/#/tokens/sessions', hash: '#/tokens/sessions', title: /^(Tokens|Token)$/, panelTitle: /^(High Token Sessions|高 Token 会话)$/ },
  { path: '/#/tools/overview', hash: '#/tools/overview', title: /^(Tools|工具)$/ },
  { path: '/#/tools/calls', hash: '#/tools/calls', title: /^(Tools|工具)$/ },
  { path: '/#/tools/shell', hash: '#/tools/shell', title: /^(Tools|工具)$/ },
  { path: '/#/audit/summary', hash: '#/audit/summary', title: /^(Audit|审计)$/ },
  { path: '/#/audit/findings', hash: '#/audit/findings', title: /^(Audit|审计)$/ },
  { path: '/#/agent-privacy', hash: '#/agent-privacy', title: /^(Agent Privacy|Agent 隐私)$/ },
  { path: '/#/settings/source', hash: '#/settings/source', title: /^(Settings|设置)$/ },
  { path: '/#/settings/display', hash: '#/settings/display', title: /^(Settings|设置)$/ }
]

const ignoredConsoleErrorPatterns = [
  /Warning: \[ant-design-vue: Typography\] When `ellipsis` is enabled, please use `content` instead of children/
]

test('key hash routes render without console errors or API 5xx responses', async ({ page }) => {
  const failures = watchPageFailures(page)

  for (const route of routes) {
    await test.step(route.hash, async () => {
      await page.goto(route.path, { waitUntil: 'domcontentloaded' })

      await expect(page).toHaveURL(new RegExp(`${escapeRegExp(route.hash)}$`))
      await expect(page.locator('.app-shell')).toBeVisible()
      await expect(page.locator('.app-content')).toBeVisible()
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

function watchPageFailures(page: Page) {
  const consoleErrors: string[] = []
  const apiResponses: string[] = []

  page.on('console', (message) => {
    if (message.type() !== 'error') return
    if (ignoredConsoleErrorPatterns.some((pattern) => pattern.test(message.text()))) return

    const location = message.location()
    const source = location.url ? ` (${location.url}:${location.lineNumber})` : ''
    consoleErrors.push(`${message.text()}${source}`)
  })

  page.on('pageerror', (error) => {
    consoleErrors.push(error.stack || error.message)
  })

  page.on('response', (response) => {
    if (!isApiUrl(response.url()) || response.status() < 500) return

    const request = response.request()
    const url = new URL(response.url())
    apiResponses.push(`${response.status()} ${request.method()} ${url.pathname}${url.search}`)
  })

  return { consoleErrors, apiResponses }
}

async function settleRoute(page: Page) {
  await page.waitForLoadState('networkidle', { timeout: 2_000 }).catch(() => undefined)
}

function isApiUrl(url: string) {
  return new URL(url).pathname.startsWith('/api')
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function formatFailures(label: string, failures: string[]) {
  if (!failures.length) return label

  return `${label}:\n${[...new Set(failures)].join('\n')}`
}
