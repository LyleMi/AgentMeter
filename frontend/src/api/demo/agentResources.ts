import type { AgentResourceOverview } from '../types'

export const agentResources: AgentResourceOverview = {
  agents: [
    {
      kind: 'codex',
      name: 'Codex',
      rootPath: 'C:\\Users\\demo\\.codex',
      exists: true,
      configPath: 'C:\\Users\\demo\\.codex\\config.toml',
      warnings: []
    }
  ],
  skills: [
    {
      agentKind: 'codex',
      name: 'frontend-design',
      title: 'Frontend Design',
      description: 'Guidance for distinctive, intentional visual design when building new UI.',
      path: 'C:\\Users\\demo\\.codex\\skills\\frontend-design',
      relativePath: 'frontend-design',
      system: false,
      sizeBytes: 8260,
      modifiedAt: '2026-06-28T08:15:00Z'
    },
    {
      agentKind: 'codex',
      name: 'openai-docs',
      title: 'OpenAI Docs',
      description: 'Use official OpenAI documentation for product and API questions.',
      path: 'C:\\Users\\demo\\.codex\\skills\\.system\\openai-docs',
      relativePath: '.system/openai-docs',
      system: true,
      sizeBytes: 18914,
      modifiedAt: '2026-06-28T08:15:00Z'
    }
  ],
  mcpServers: [
    {
      agentKind: 'codex',
      name: 'node_repl',
      command: 'node_repl.exe',
      args: [],
      envKeys: ['NODE_OPTIONS'],
      configPath: 'C:\\Users\\demo\\.codex\\config.toml',
      enabled: true,
      status: 'configured'
    }
  ],
  memories: [
    {
      agentKind: 'codex',
      name: 'MEMORY',
      title: 'Memory',
      path: 'C:\\Users\\demo\\.codex\\memories\\MEMORY.md',
      relativePath: 'MEMORY.md',
      kind: 'primary',
      preview: 'Keep responses concise, direct, and grounded in the local repository.',
      sizeBytes: 10256,
      modifiedAt: '2026-07-04T09:30:00Z'
    },
    {
      agentKind: 'codex',
      name: 'memory_summary',
      title: 'Memory Summary',
      path: 'C:\\Users\\demo\\.codex\\memories\\memory_summary.md',
      relativePath: 'memory_summary.md',
      kind: 'summary',
      preview: 'Current long-running work and preferences summarized for future sessions.',
      sizeBytes: 3740,
      modifiedAt: '2026-07-04T09:32:00Z'
    }
  ],
  warnings: []
}
