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
    },
    {
      kind: 'gemini',
      name: 'Gemini CLI',
      rootPath: 'C:\\Users\\demo\\.gemini',
      exists: true,
      configPath: 'C:\\Users\\demo\\.gemini\\settings.json',
      warnings: []
    }
  ],
  skills: [
    {
      agentKind: 'codex',
      resourceType: 'skill',
      name: 'frontend-design',
      title: 'Frontend Design',
      description: 'Guidance for distinctive, intentional visual design when building new UI.',
      path: 'C:\\Users\\demo\\.codex\\skills\\frontend-design',
      relativePath: 'frontend-design',
      system: false,
      enabled: true,
      canToggle: true,
      status: 'enabled',
      sizeBytes: 8260,
      modifiedAt: '2026-06-28T08:15:00Z'
    },
    {
      agentKind: 'codex',
      resourceType: 'skill',
      name: 'openai-docs',
      title: 'OpenAI Docs',
      description: 'Use official OpenAI documentation for product and API questions.',
      path: 'C:\\Users\\demo\\.codex\\skills\\.system\\openai-docs',
      relativePath: '.system/openai-docs',
      system: true,
      enabled: true,
      canToggle: false,
      status: 'enabled',
      sizeBytes: 18914,
      modifiedAt: '2026-06-28T08:15:00Z'
    },
    {
      agentKind: 'gemini',
      resourceType: 'skill',
      name: 'workspace-context',
      title: 'Workspace Context',
      description: 'Project-scoped Gemini CLI instructions and context files.',
      path: 'C:\\Users\\demo\\.gemini\\skills\\workspace-context',
      relativePath: 'workspace-context',
      system: false,
      enabled: false,
      canToggle: true,
      status: 'disabled',
      sizeBytes: 4212,
      modifiedAt: '2026-07-03T10:20:00Z'
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
      canToggle: true,
      status: 'configured'
    },
    {
      agentKind: 'gemini',
      name: 'filesystem',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-filesystem'],
      envKeys: [],
      configPath: 'C:\\Users\\demo\\.gemini\\settings.json',
      enabled: false,
      canToggle: true,
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
      content: 'Keep responses concise, direct, and grounded in the local repository.',
      canEdit: true,
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
      content: 'Current long-running work and preferences summarized for future sessions.',
      canEdit: true,
      sizeBytes: 3740,
      modifiedAt: '2026-07-04T09:32:00Z'
    },
    {
      agentKind: 'gemini',
      name: 'GEMINI',
      title: 'Gemini Memory',
      path: 'C:\\Users\\demo\\.gemini\\GEMINI.md',
      relativePath: 'GEMINI.md',
      kind: 'primary',
      preview: 'Prefer project-local context and avoid global assumptions.',
      content: 'Prefer project-local context and avoid global assumptions.',
      canEdit: true,
      sizeBytes: 2180,
      modifiedAt: '2026-07-04T09:35:00Z'
    }
  ],
  warnings: []
}
