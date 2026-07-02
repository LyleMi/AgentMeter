import type { DemoSource } from './contracts'

export const sources: DemoSource[] = [
  {
    sourceId: 1,
    sourceKey: 'source:1',
    sourceLabel: 'Codex CLI',
    sourceRootPath: 'C:\\Users\\demo\\.codex',
    sourceSessionsPath: 'C:\\Users\\demo\\.codex\\sessions',
    agentKind: 'codex',
    agentName: 'Codex CLI'
  },
  {
    sourceId: 2,
    sourceKey: 'source:2',
    sourceLabel: 'Gemini CLI',
    sourceRootPath: 'C:\\Users\\demo\\.gemini',
    sourceSessionsPath: 'C:\\Users\\demo\\.gemini\\tmp',
    agentKind: 'gemini',
    agentName: 'Gemini CLI'
  },
  {
    sourceId: 3,
    sourceKey: 'source:3',
    sourceLabel: 'Claude Code',
    sourceRootPath: 'C:\\Users\\demo\\.claude',
    sourceSessionsPath: 'C:\\Users\\demo\\.claude\\projects',
    agentKind: 'claude',
    agentName: 'Claude Code'
  }
]

export function source(index: number): DemoSource {
  return sources[index]
}
