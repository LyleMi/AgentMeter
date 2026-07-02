import type { IndexResult, Settings, SourceEntry } from '../types'
import { pricingModels } from './pricing'
import { sessions } from './sessions'
import { sources } from './sources'

export function settings(sourceEntries: SourceEntry[] = sources.map((item) => ({ path: item.sourceRootPath, enabled: true, label: item.sourceLabel }))): Settings {
  const result = indexResult(false)
  const enabledEntries = sourceEntries.filter((entry) => entry.enabled)
  return {
    sourcePath: enabledEntries[0]?.path || '',
    sourcePaths: enabledEntries.map((entry) => entry.path),
    sourceEntries,
    defaultSourcePath: sources[0].sourceRootPath,
    defaultSourcePaths: sources.map((item) => item.sourceRootPath),
    databasePath: 'C:\\Users\\demo\\AppData\\Local\\AgentMeter\\agentmeter-demo.db',
    pricingModels,
    lastIndexStartedAt: '2026-06-28T02:00:00Z',
    lastIndexResult: result
  }
}

export function indexResult(rebuild: boolean): IndexResult {
  return {
    sourcePath: sources[0].sourceRootPath,
    sourcePaths: sources.map((item) => item.sourceRootPath),
    database: 'C:\\Users\\demo\\AppData\\Local\\AgentMeter\\agentmeter-demo.db',
    filesSeen: 18,
    indexed: sessions.length,
    skipped: 2,
    failed: 0,
    sessions: sessions.length,
    warnings: ['Static demo mode is read-only. Index requests are simulated and no files are scanned.'],
    durationMs: rebuild ? 1420 : 460,
    rebuild
  }
}
