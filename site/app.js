const agents = [
  { name: 'Codex', sessions: 44, tokens: '1.52M', cost: '$13.84', share: 32 },
  { name: 'Claude Code', sessions: 31, tokens: '1.07M', cost: '$9.18', share: 22 },
  { name: 'CodeBuddy', sessions: 27, tokens: '912K', cost: '$7.42', share: 19 },
  { name: 'WorkBuddy', sessions: 24, tokens: '743K', cost: '$6.31', share: 15 },
  { name: 'Generic JSONL', sessions: 20, tokens: '578K', cost: '$4.52', share: 12 }
]

const sessions = [
  {
    name: 'pricing-index-regression',
    agent: 'Codex',
    model: 'gpt-5-codex',
    tokens: '184K',
    cost: '$1.62',
    wall: '42m 14s',
    tools: 47,
    status: 'review',
    kind: 'long'
  },
  {
    name: 'agent-privacy-profile',
    agent: 'Claude Code',
    model: 'claude-sonnet-4',
    tokens: '126K',
    cost: '$1.10',
    wall: '31m 08s',
    tools: 36,
    status: 'ok',
    kind: 'review'
  },
  {
    name: 'frontend-smoke-hmr',
    agent: 'CodeBuddy',
    model: 'codebuddy-pro',
    tokens: '91K',
    cost: '$0.73',
    wall: '18m 52s',
    tools: 24,
    status: 'ok',
    kind: 'all'
  },
  {
    name: 'jsonl-parser-fixtures',
    agent: 'WorkBuddy',
    model: 'workbuddy-coder',
    tokens: '77K',
    cost: '$0.58',
    wall: '22m 39s',
    tools: 19,
    status: 'ok',
    kind: 'all'
  },
  {
    name: 'legacy-source-import',
    agent: 'Generic JSONL',
    model: 'unknown-jsonl',
    tokens: '64K',
    cost: '$0.41',
    wall: '12m 25s',
    tools: 15,
    status: 'unpriced',
    kind: 'review'
  },
  {
    name: 'audit-command-review',
    agent: 'Codex',
    model: 'gpt-5-codex',
    tokens: '138K',
    cost: '$1.21',
    wall: '36m 18s',
    tools: 41,
    status: 'review',
    kind: 'long'
  }
]

const tools = [
  { name: 'rg', group: 'search', calls: 286, duration: '23m 18s', share: 76 },
  { name: 'apply_patch', group: 'edit', calls: 174, duration: '18m 04s', share: 58 },
  { name: 'go test', group: 'validation', calls: 91, duration: '41m 12s', share: 44 },
  { name: 'npm run build', group: 'validation', calls: 37, duration: '28m 45s', share: 31 },
  { name: 'browser smoke', group: 'verification', calls: 22, duration: '14m 20s', share: 18 }
]

const findings = [
  {
    severity: 'warning',
    title: 'Broad shell command needs review',
    copy: 'A mock Codex session ran a recursive command over a large workspace before narrowing scope.',
    source: 'Codex'
  },
  {
    severity: 'warning',
    title: 'Potential network fetch in tool trace',
    copy: 'A CodeBuddy tool call matched the network-likely heuristic for package metadata lookup.',
    source: 'CodeBuddy'
  },
  {
    severity: 'info',
    title: 'Generic JSONL source missing model price',
    copy: 'Unknown model records were indexed and left out of exact cost totals in the sample session.',
    source: 'Generic JSONL'
  },
  {
    severity: 'info',
    title: 'Privacy profile records are static',
    copy: 'The public preview shows representative statuses and does not inspect real agent config files.',
    source: 'Privacy'
  }
]

const viewLinks = Array.from(document.querySelectorAll('[data-view-link]'))
const views = Array.from(document.querySelectorAll('[data-view]'))
const agentList = document.querySelector('[data-agent-list]')
const sessionTable = document.querySelector('[data-session-table]')
const toolList = document.querySelector('[data-tool-list]')
const findingList = document.querySelector('[data-finding-list]')
const densityToggle = document.querySelector('[data-density-toggle]')

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;')
}

function setView(name) {
  const target = views.some((view) => view.dataset.view === name) ? name : 'overview'

  views.forEach((view) => {
    view.classList.toggle('is-active', view.dataset.view === target)
  })

  viewLinks.forEach((link) => {
    const isCurrent = link.dataset.viewLink === target
    if (isCurrent) {
      link.setAttribute('aria-current', 'page')
    } else {
      link.removeAttribute('aria-current')
    }
  })
}

function renderAgents() {
  agentList.innerHTML = agents
    .map(
      (agent) => `
        <div class="agent-row">
          <div>
            <span class="agent-name">${escapeHtml(agent.name)}</span>
            <span class="agent-meta">${agent.sessions} sessions / ${agent.cost}</span>
          </div>
          <div class="meter" aria-label="${escapeHtml(agent.name)} token share ${agent.share}%">
            <span style="--value: ${agent.share}%"></span>
          </div>
          <span class="row-value">${agent.tokens}</span>
        </div>
      `
    )
    .join('')
}

function statusTag(status) {
  if (status === 'ok') return '<span class="tag success">ok</span>'
  if (status === 'unpriced') return '<span class="tag warning">unpriced</span>'
  return '<span class="tag info">review</span>'
}

function renderSessions(filter = 'all') {
  const filtered = sessions.filter((session) => {
    if (filter === 'all') return true
    if (filter === 'long') return session.kind === 'long'
    return session.status === 'review' || session.kind === 'review'
  })

  sessionTable.innerHTML = filtered
    .map(
      (session) => `
        <tr>
          <td><span class="mono">${escapeHtml(session.name)}</span></td>
          <td>${escapeHtml(session.agent)}</td>
          <td>${escapeHtml(session.model)}</td>
          <td class="mono">${escapeHtml(session.tokens)}</td>
          <td class="mono">${escapeHtml(session.cost)}</td>
          <td>${escapeHtml(session.wall)}</td>
          <td>${session.tools}</td>
          <td>${statusTag(session.status)}</td>
        </tr>
      `
    )
    .join('')
}

function renderTools() {
  toolList.innerHTML = tools
    .map(
      (tool) => `
        <div class="tool-row">
          <div>
            <span class="tool-name">${escapeHtml(tool.name)}</span>
            <span class="tool-meta">${escapeHtml(tool.group)} / ${escapeHtml(tool.duration)}</span>
          </div>
          <div class="meter" aria-label="${escapeHtml(tool.name)} relative call volume ${tool.share}%">
            <span style="--value: ${tool.share}%"></span>
          </div>
          <span class="row-value">${tool.calls}</span>
        </div>
      `
    )
    .join('')
}

function renderFindings() {
  findingList.innerHTML = findings
    .map(
      (finding) => `
        <div class="finding-item">
          <span class="finding-severity ${finding.severity}" aria-hidden="true"></span>
          <div>
            <div class="finding-title">${escapeHtml(finding.title)}</div>
            <div class="finding-copy">${escapeHtml(finding.copy)}</div>
          </div>
          <span class="tag ${finding.severity}">${escapeHtml(finding.source)}</span>
        </div>
      `
    )
    .join('')
}

window.addEventListener('hashchange', () => {
  setView(window.location.hash.slice(1))
})

viewLinks.forEach((link) => {
  link.addEventListener('click', () => {
    setView(link.dataset.viewLink)
  })
})

document.querySelectorAll('[data-filter]').forEach((button) => {
  button.addEventListener('click', () => {
    document.querySelectorAll('[data-filter]').forEach((item) => {
      item.classList.toggle('is-selected', item === button)
    })
    renderSessions(button.dataset.filter)
  })
})

densityToggle.addEventListener('click', () => {
  document.body.classList.toggle('is-compact')
  densityToggle.textContent = document.body.classList.contains('is-compact') ? 'Comfortable' : 'Compact'
})

renderAgents()
renderSessions()
renderTools()
renderFindings()
setView(window.location.hash.slice(1))
