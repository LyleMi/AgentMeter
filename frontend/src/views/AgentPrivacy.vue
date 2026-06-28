<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import { api } from '../api/client'
import type { PrivacyConfigApplyResult, PrivacyConfigSetting, PrivacyConfigStatus, PrivacyTarget } from '../api/types'
import PrivacySettingsPanel from '../components/privacy/PrivacySettingsPanel.vue'
import PrivacySummaryPanel from '../components/privacy/PrivacySummaryPanel.vue'
import { useAgentPrivacyEditor } from '../composables/useAgentPrivacyEditor'
import { useMessages } from '../i18n'
import { formatNumber } from '../presentation/formatters'
import {
  formatPrivacyConfigValue,
  privacyValueType,
  privacyValueTypeLabel,
  privacyValuesEqual,
  strictPrivacyValue
} from '../presentation/privacyConfig'

const { t, locale } = useMessages({
  en: {
    'privacy.title': 'Agent Privacy',
    'privacy.kicker': 'External agent config: {target} user-level {file}',
    'privacy.boundary.title': 'Current support: user-level agent config controls',
    'privacy.boundary.description':
      'This page reads and edits supported user-level privacy settings for Codex, Gemini CLI, Claude Code, and CodeBuddy Code/IDE. It does not scan logs, scan secrets, or infer broad filesystem policy.',
    'privacy.target.codex': 'Codex',
    'privacy.target.gemini': 'Gemini CLI',
    'privacy.target.claude': 'Claude Code',
    'privacy.target.codebuddy': 'CodeBuddy',
    'privacy.action.refresh': 'Refresh',
    'privacy.action.saveAll': 'Save changes',
    'privacy.action.useStrict': 'Use strict',
    'privacy.action.unset': 'Unset',
    'privacy.action.reset': 'Reset',
    'privacy.action.save': 'Save',
    'privacy.meta.target': 'Target',
    'privacy.meta.configPath': 'Config path',
    'privacy.meta.file': 'File',
    'privacy.meta.exists': 'Exists',
    'privacy.meta.missing': 'Missing',
    'privacy.meta.total': 'Total',
    'privacy.meta.strictConfigured': 'Strict',
    'privacy.meta.defaultSafe': 'Default-safe',
    'privacy.meta.customConfigured': 'Custom configured',
    'privacy.meta.missingRequired': 'Unset',
    'privacy.meta.unsavedChanges': 'Unsaved',
    'privacy.meta.backupPath': 'Backup path',
    'privacy.status.ready': 'No review needed',
    'privacy.status.needsChange': 'Needs review',
    'privacy.status.noStatus': 'No status',
    'privacy.status.hardened': 'strict',
    'privacy.status.implicit': 'default-safe',
    'privacy.status.attention': 'needs review',
    'privacy.status.unknown': 'unknown',
    'privacy.settings.title': 'Configuration groups',
    'privacy.settings.kicker': 'Edit current values directly, then save individual rows or all changes',
    'privacy.empty': 'No privacy controls returned',
    'privacy.value.unset': 'unset',
    'privacy.value.notConfigured': 'not configured',
    'privacy.value.pendingUnset': 'will unset on save',
    'privacy.value.type': 'Type',
    'privacy.value.current': 'Current',
    'privacy.value.editable': 'Editable',
    'privacy.value.strict': 'Strict',
    'privacy.value.unsaved': 'Unsaved',
    'privacy.message.loadFailed': 'Load privacy settings failed',
    'privacy.message.saveFailed': 'Save privacy settings failed',
    'privacy.message.targetMismatch': 'Privacy API returned a different target',
    'privacy.message.saved': 'Saved {count} changes',
    'privacy.message.noChanges': 'No unsaved changes',
    'privacy.result.title': 'Last save result',
    'privacy.result.changed': 'Changed',
    'privacy.result.noChanges': 'No config values changed',
    'privacy.warning.title': 'Warnings',
    'privacy.group.default': 'General',
    'privacy.settingGroup.telemetry': 'Telemetry',
    'privacy.settingGroup.network': 'Network',
    'privacy.settingGroup.localHistory': 'Local history',
    'privacy.settingGroup.memory': 'Memory',
    'privacy.settingGroup.environment': 'Environment',
    'privacy.settingGroup.usage': 'Usage',
    'privacy.settingGroup.localRetention': 'Local retention',
    'privacy.settingGroup.approval': 'Approval',
    'privacy.settingGroup.extensions': 'Extensions',
    'privacy.settingGroup.browser': 'Browser',
    'privacy.settingGroup.voice': 'Voice',
    'privacy.setting.codex.analytics.enabled.title': 'Analytics',
    'privacy.setting.codex.analytics.enabled.description': 'Disables Codex analytics collection in the user config.',
    'privacy.setting.codex.analytics.enabled.impact': 'Keeps analytics disabled explicitly.',
    'privacy.setting.codex.otel.exporter.title': 'OpenTelemetry exporter',
    'privacy.setting.codex.otel.exporter.description': 'Disables the general OpenTelemetry exporter.',
    'privacy.setting.codex.otel.exporter.impact': 'Prevents telemetry export from the Codex process.',
    'privacy.setting.codex.otel.trace_exporter.title': 'Trace exporter',
    'privacy.setting.codex.otel.trace_exporter.description': 'Disables OpenTelemetry trace export.',
    'privacy.setting.codex.otel.trace_exporter.impact': 'Prevents trace spans from leaving the machine.',
    'privacy.setting.codex.otel.metrics_exporter.title': 'Metrics exporter',
    'privacy.setting.codex.otel.metrics_exporter.description': 'Disables OpenTelemetry metrics export.',
    'privacy.setting.codex.otel.metrics_exporter.impact': 'Prevents metrics export from the Codex process.',
    'privacy.setting.codex.otel.log_user_prompt.title': 'Prompt logging',
    'privacy.setting.codex.otel.log_user_prompt.description': 'Keeps user prompt logging disabled for telemetry.',
    'privacy.setting.codex.otel.log_user_prompt.impact': 'Avoids including prompt text in telemetry logs.',
    'privacy.setting.codex.web_search.title': 'Web search',
    'privacy.setting.codex.web_search.description': 'Disables Codex web search from user config.',
    'privacy.setting.codex.web_search.impact': 'Keeps prompts and search queries from using web search by default.',
    'privacy.setting.codex.history.persistence.title': 'Conversation history',
    'privacy.setting.codex.history.persistence.description': 'Disables local Codex history persistence.',
    'privacy.setting.codex.history.persistence.impact': 'Reduces local retention of prompts and responses.',
    'privacy.setting.codex.features.memories.title': 'Memory feature',
    'privacy.setting.codex.features.memories.description': 'Disables the Codex memory feature.',
    'privacy.setting.codex.features.memories.impact': 'Keeps durable memory features disabled explicitly.',
    'privacy.setting.codex.memories.generate_memories.title': 'Generate memories',
    'privacy.setting.codex.memories.generate_memories.description': 'Prevents Codex from generating memories.',
    'privacy.setting.codex.memories.generate_memories.impact': 'Avoids creating durable memory records from conversations.',
    'privacy.setting.codex.memories.use_memories.title': 'Use memories',
    'privacy.setting.codex.memories.use_memories.description': 'Prevents Codex from using saved memories.',
    'privacy.setting.codex.memories.use_memories.impact': 'Avoids injecting saved memories into future context.',
    'privacy.setting.codex.memories.disable_on_external_context.title': 'External context memory guard',
    'privacy.setting.codex.memories.disable_on_external_context.description':
      'Keeps memories disabled when external context is present.',
    'privacy.setting.codex.memories.disable_on_external_context.impact':
      'Reduces memory use when outside context may be present.',
    'privacy.setting.codex.sandbox_workspace_write.network_access.title': 'Workspace network access',
    'privacy.setting.codex.sandbox_workspace_write.network_access.description':
      'Disables network access for workspace-write sandbox mode.',
    'privacy.setting.codex.sandbox_workspace_write.network_access.impact':
      'Keeps sandboxed commands offline unless explicitly changed later.',
    'privacy.setting.codex.shell_environment_policy.inherit.title': 'Shell environment inheritance',
    'privacy.setting.codex.shell_environment_policy.inherit.description':
      'Limits inherited shell environment variables to Codex core defaults.',
    'privacy.setting.codex.shell_environment_policy.inherit.impact':
      'Reduces accidental exposure of environment variables to shell commands.',
    'privacy.setting.codex.shell_environment_policy.ignore_default_excludes.title': 'Default environment excludes',
    'privacy.setting.codex.shell_environment_policy.ignore_default_excludes.description':
      'Keeps Codex default environment-variable excludes active.',
    'privacy.setting.codex.shell_environment_policy.ignore_default_excludes.impact':
      'Preserves default filtering for sensitive environment variables.',
    'privacy.setting.gemini.privacy.usageStatisticsEnabled.title': 'Usage statistics',
    'privacy.setting.gemini.privacy.usageStatisticsEnabled.description':
      'Opts out of Gemini CLI usage statistics collection.',
    'privacy.setting.gemini.privacy.usageStatisticsEnabled.impact':
      'Prevents Gemini CLI usage statistics from being sent to Google.',
    'privacy.setting.gemini.telemetry.enabled.title': 'OpenTelemetry',
    'privacy.setting.gemini.telemetry.enabled.description': 'Disables Gemini CLI OpenTelemetry emission.',
    'privacy.setting.gemini.telemetry.enabled.impact':
      'Prevents telemetry logs, metrics, and traces from being exported.',
    'privacy.setting.gemini.telemetry.traces.title': 'Detailed traces',
    'privacy.setting.gemini.telemetry.traces.description': 'Disables detailed telemetry traces.',
    'privacy.setting.gemini.telemetry.traces.impact':
      'Avoids detailed attributes such as tool output and file-read trace data.',
    'privacy.setting.gemini.telemetry.logPrompts.title': 'Prompt logging',
    'privacy.setting.gemini.telemetry.logPrompts.description':
      'Prevents prompts from being included in telemetry logs.',
    'privacy.setting.gemini.telemetry.logPrompts.impact':
      'Keeps prompt text out of telemetry if telemetry is enabled later.',
    'privacy.setting.gemini.general.logRagSnippets.title': 'RAG snippet logging',
    'privacy.setting.gemini.general.logRagSnippets.description': 'Disables local logging of full RAG snippets.',
    'privacy.setting.gemini.general.logRagSnippets.impact':
      'Avoids writing retrieved code customization snippets to local debug logs.',
    'privacy.setting.gemini.general.checkpointing.enabled.title': 'Session checkpointing',
    'privacy.setting.gemini.general.checkpointing.enabled.description': 'Disables session checkpointing.',
    'privacy.setting.gemini.general.checkpointing.enabled.impact': 'Avoids extra recovery snapshots of working state.',
    'privacy.setting.gemini.general.sessionRetention.enabled.title': 'Session cleanup',
    'privacy.setting.gemini.general.sessionRetention.enabled.description': 'Keeps automatic session cleanup enabled.',
    'privacy.setting.gemini.general.sessionRetention.enabled.impact':
      'Ensures old Gemini CLI chats are eligible for automatic cleanup.',
    'privacy.setting.gemini.general.sessionRetention.maxAge.title': 'Chat retention window',
    'privacy.setting.gemini.general.sessionRetention.maxAge.description':
      'Reduces the Gemini CLI chat retention window.',
    'privacy.setting.gemini.general.sessionRetention.maxAge.impact':
      'Keeps fewer local chat records than the default 30 day retention window.',
    'privacy.setting.gemini.tools.sandboxNetworkAccess.title': 'Sandbox network access',
    'privacy.setting.gemini.tools.sandboxNetworkAccess.description':
      'Disables network access inside the Gemini CLI sandbox.',
    'privacy.setting.gemini.tools.sandboxNetworkAccess.impact':
      'Keeps sandboxed tool execution offline unless explicitly changed later.',
    'privacy.setting.gemini.tools.exclude.web.title': 'Web tools',
    'privacy.setting.gemini.tools.exclude.web.description': 'Excludes Gemini CLI web search and web fetch tools.',
    'privacy.setting.gemini.tools.exclude.web.impact':
      'Prevents the model from using built-in web search or URL fetch tools by default.',
    'privacy.setting.gemini.experimental.directWebFetch.title': 'Direct web fetch',
    'privacy.setting.gemini.experimental.directWebFetch.description': 'Disables direct web fetch behavior.',
    'privacy.setting.gemini.experimental.directWebFetch.impact':
      'Avoids web fetch paths that bypass LLM summarization.',
    'privacy.setting.gemini.advanced.ignoreLocalEnv.title': 'Local .env loading',
    'privacy.setting.gemini.advanced.ignoreLocalEnv.description': 'Ignores generic project .env files.',
    'privacy.setting.gemini.advanced.ignoreLocalEnv.impact':
      'Reduces accidental loading of project secrets into the Gemini CLI process.',
    'privacy.setting.gemini.security.environmentVariableRedaction.enabled.title': 'Environment variable redaction',
    'privacy.setting.gemini.security.environmentVariableRedaction.enabled.description':
      'Enables redaction for sensitive environment variables.',
    'privacy.setting.gemini.security.environmentVariableRedaction.enabled.impact':
      'Redacts environment variables that may contain secrets.',
    'privacy.setting.gemini.security.disableYoloMode.title': 'YOLO mode',
    'privacy.setting.gemini.security.disableYoloMode.description': 'Disables YOLO mode even when requested by flag.',
    'privacy.setting.gemini.security.disableYoloMode.impact':
      'Prevents broad automatic approval from being enabled accidentally.',
    'privacy.setting.gemini.security.disableAlwaysAllow.title': 'Always allow',
    'privacy.setting.gemini.security.disableAlwaysAllow.description': 'Disables persistent Always allow choices.',
    'privacy.setting.gemini.security.disableAlwaysAllow.impact':
      'Reduces long-lived tool approvals that can leak into future sessions.',
    'privacy.setting.gemini.security.enablePermanentToolApproval.title': 'Permanent tool approval',
    'privacy.setting.gemini.security.enablePermanentToolApproval.description': 'Disables permanent tool approval.',
    'privacy.setting.gemini.security.enablePermanentToolApproval.impact':
      'Avoids future-session approvals being added from confirmation dialogs.',
    'privacy.setting.gemini.security.blockGitExtensions.title': 'Git extensions',
    'privacy.setting.gemini.security.blockGitExtensions.description':
      'Blocks installing and loading extensions from Git.',
    'privacy.setting.gemini.security.blockGitExtensions.impact':
      'Reduces exposure to remote extension code and extension-provided tools.',
    'privacy.setting.gemini.agents.browser.confirmSensitiveActions.title': 'Sensitive browser actions',
    'privacy.setting.gemini.agents.browser.confirmSensitiveActions.description':
      'Requires confirmation for sensitive browser actions.',
    'privacy.setting.gemini.agents.browser.confirmSensitiveActions.impact':
      'Requires manual confirmation before filling forms or running browser scripts.',
    'privacy.setting.gemini.agents.browser.blockFileUploads.title': 'Browser file uploads',
    'privacy.setting.gemini.agents.browser.blockFileUploads.description': 'Blocks file uploads from the browser agent.',
    'privacy.setting.gemini.agents.browser.blockFileUploads.impact':
      'Prevents browser automation from uploading local files.',
    'privacy.setting.gemini.experimental.voiceMode.title': 'Voice mode',
    'privacy.setting.gemini.experimental.voiceMode.description': 'Disables experimental voice mode.',
    'privacy.setting.gemini.experimental.voiceMode.impact':
      'Avoids voice workflows that may send recordings to a cloud transcription backend.',
    'privacy.setting.gemini.experimental.autoMemory.title': 'Auto Memory',
    'privacy.setting.gemini.experimental.autoMemory.description':
      'Disables automatic memory and skill extraction from past sessions.',
    'privacy.setting.gemini.experimental.autoMemory.impact':
      'Prevents background model calls over selected local transcript content.',
    'privacy.setting.gemini.context.loadMemoryFromIncludeDirectories.title': 'Include-directory memory',
    'privacy.setting.gemini.context.loadMemoryFromIncludeDirectories.description':
      'Disables loading memory files from include directories.',
    'privacy.setting.gemini.context.loadMemoryFromIncludeDirectories.impact':
      'Keeps /memory reload scoped to the current directory by default.',
    'privacy.setting.gemini.skills.enabled.title': 'Agent skills',
    'privacy.setting.gemini.skills.enabled.description': 'Disables Gemini CLI agent skills.',
    'privacy.setting.gemini.skills.enabled.impact':
      'Avoids injecting local skill instructions into future agent context.'
  },
  'zh-CN': {
    'privacy.title': 'Agent 隐私',
    'privacy.kicker': '外部 Agent 配置：{target} 用户级 {file}',
    'privacy.boundary.title': '当前支持范围：用户级 Agent 配置控制项',
    'privacy.boundary.description':
      '此页面只读取并编辑 Codex、Gemini CLI、Claude Code 与 CodeBuddy Code/IDE 已支持的用户级隐私设置，不扫描日志、不扫描密钥，也不推断广义文件系统策略。',
    'privacy.target.codex': 'Codex',
    'privacy.target.gemini': 'Gemini CLI',
    'privacy.target.claude': 'Claude Code',
    'privacy.target.codebuddy': 'CodeBuddy',
    'privacy.action.refresh': '刷新',
    'privacy.action.saveAll': '保存变更',
    'privacy.action.useStrict': '使用严格值',
    'privacy.action.unset': '取消设置',
    'privacy.action.reset': '重置',
    'privacy.action.save': '保存',
    'privacy.meta.target': '目标',
    'privacy.meta.configPath': '配置路径',
    'privacy.meta.file': '文件',
    'privacy.meta.exists': '存在',
    'privacy.meta.missing': '缺失',
    'privacy.meta.total': '总数',
    'privacy.meta.strictConfigured': '严格值',
    'privacy.meta.defaultSafe': '默认安全',
    'privacy.meta.customConfigured': '显式自定义',
    'privacy.meta.missingRequired': '未设置',
    'privacy.meta.unsavedChanges': '未保存',
    'privacy.meta.backupPath': '备份路径',
    'privacy.status.ready': '无需检查',
    'privacy.status.needsChange': '需要检查',
    'privacy.status.noStatus': '无状态',
    'privacy.status.hardened': '严格值',
    'privacy.status.implicit': '默认安全',
    'privacy.status.attention': '需要检查',
    'privacy.status.unknown': '未知',
    'privacy.settings.title': '配置分组',
    'privacy.settings.kicker': '直接编辑当前值，再保存单行或所有变更',
    'privacy.empty': '未返回隐私控制项',
    'privacy.value.unset': '未设置',
    'privacy.value.notConfigured': '未配置',
    'privacy.value.pendingUnset': '保存后取消设置',
    'privacy.value.type': '类型',
    'privacy.value.current': '当前',
    'privacy.value.editable': '编辑值',
    'privacy.value.strict': '严格值',
    'privacy.value.unsaved': '未保存',
    'privacy.message.loadFailed': '加载隐私设置失败',
    'privacy.message.saveFailed': '保存隐私设置失败',
    'privacy.message.targetMismatch': '隐私 API 返回了不同目标',
    'privacy.message.saved': '已保存 {count} 项变更',
    'privacy.message.noChanges': '没有未保存变更',
    'privacy.result.title': '上次保存结果',
    'privacy.result.changed': '已变更',
    'privacy.result.noChanges': '没有配置值变更',
    'privacy.warning.title': '警告',
    'privacy.group.default': '通用',
    'privacy.settingGroup.telemetry': '遥测',
    'privacy.settingGroup.network': '网络',
    'privacy.settingGroup.localHistory': '本地历史',
    'privacy.settingGroup.memory': '记忆',
    'privacy.settingGroup.environment': '环境',
    'privacy.settingGroup.usage': '使用情况',
    'privacy.settingGroup.localRetention': '本地保留',
    'privacy.settingGroup.approval': '审批',
    'privacy.settingGroup.extensions': '扩展',
    'privacy.settingGroup.browser': '浏览器',
    'privacy.settingGroup.voice': '语音',
    'privacy.setting.codex.analytics.enabled.title': '分析统计',
    'privacy.setting.codex.analytics.enabled.description': '在用户配置中禁用 Codex 分析数据收集。',
    'privacy.setting.codex.analytics.enabled.impact': '明确保持分析统计为禁用状态。',
    'privacy.setting.codex.otel.exporter.title': 'OpenTelemetry 导出器',
    'privacy.setting.codex.otel.exporter.description': '禁用通用 OpenTelemetry 导出器。',
    'privacy.setting.codex.otel.exporter.impact': '阻止 Codex 进程导出遥测数据。',
    'privacy.setting.codex.otel.trace_exporter.title': '追踪导出器',
    'privacy.setting.codex.otel.trace_exporter.description': '禁用 OpenTelemetry 追踪导出。',
    'privacy.setting.codex.otel.trace_exporter.impact': '阻止追踪 span 离开本机。',
    'privacy.setting.codex.otel.metrics_exporter.title': '指标导出器',
    'privacy.setting.codex.otel.metrics_exporter.description': '禁用 OpenTelemetry 指标导出。',
    'privacy.setting.codex.otel.metrics_exporter.impact': '阻止 Codex 进程导出指标。',
    'privacy.setting.codex.otel.log_user_prompt.title': '提示词日志',
    'privacy.setting.codex.otel.log_user_prompt.description': '保持遥测中的用户提示词日志关闭。',
    'privacy.setting.codex.otel.log_user_prompt.impact': '避免在遥测日志中包含提示词文本。',
    'privacy.setting.codex.web_search.title': 'Web 搜索',
    'privacy.setting.codex.web_search.description': '通过用户配置禁用 Codex Web 搜索。',
    'privacy.setting.codex.web_search.impact': '默认阻止提示词和搜索查询使用 Web 搜索。',
    'privacy.setting.codex.history.persistence.title': '对话历史',
    'privacy.setting.codex.history.persistence.description': '禁用本地 Codex 历史持久化。',
    'privacy.setting.codex.history.persistence.impact': '减少提示词和回复的本地保留。',
    'privacy.setting.codex.features.memories.title': '记忆功能',
    'privacy.setting.codex.features.memories.description': '禁用 Codex 记忆功能。',
    'privacy.setting.codex.features.memories.impact': '明确保持持久记忆功能禁用。',
    'privacy.setting.codex.memories.generate_memories.title': '生成记忆',
    'privacy.setting.codex.memories.generate_memories.description': '阻止 Codex 生成记忆。',
    'privacy.setting.codex.memories.generate_memories.impact': '避免从对话创建持久记忆记录。',
    'privacy.setting.codex.memories.use_memories.title': '使用记忆',
    'privacy.setting.codex.memories.use_memories.description': '阻止 Codex 使用已保存的记忆。',
    'privacy.setting.codex.memories.use_memories.impact': '避免将已保存记忆注入后续上下文。',
    'privacy.setting.codex.memories.disable_on_external_context.title': '外部上下文记忆保护',
    'privacy.setting.codex.memories.disable_on_external_context.description': '存在外部上下文时保持记忆禁用。',
    'privacy.setting.codex.memories.disable_on_external_context.impact': '在可能存在外部上下文时减少记忆使用。',
    'privacy.setting.codex.sandbox_workspace_write.network_access.title': '工作区网络访问',
    'privacy.setting.codex.sandbox_workspace_write.network_access.description':
      '禁用 workspace-write 沙盒模式的网络访问。',
    'privacy.setting.codex.sandbox_workspace_write.network_access.impact':
      '除非之后显式更改，否则让沙盒命令保持离线。',
    'privacy.setting.codex.shell_environment_policy.inherit.title': 'Shell 环境继承',
    'privacy.setting.codex.shell_environment_policy.inherit.description':
      '将继承的 Shell 环境变量限制为 Codex 核心默认值。',
    'privacy.setting.codex.shell_environment_policy.inherit.impact': '减少环境变量意外暴露给 Shell 命令。',
    'privacy.setting.codex.shell_environment_policy.ignore_default_excludes.title': '默认环境排除项',
    'privacy.setting.codex.shell_environment_policy.ignore_default_excludes.description':
      '保持 Codex 默认环境变量排除项生效。',
    'privacy.setting.codex.shell_environment_policy.ignore_default_excludes.impact':
      '保留对敏感环境变量的默认过滤。',
    'privacy.setting.gemini.privacy.usageStatisticsEnabled.title': '使用统计',
    'privacy.setting.gemini.privacy.usageStatisticsEnabled.description': '退出 Gemini CLI 使用统计收集。',
    'privacy.setting.gemini.privacy.usageStatisticsEnabled.impact': '阻止 Gemini CLI 使用统计发送到 Google。',
    'privacy.setting.gemini.telemetry.enabled.title': 'OpenTelemetry',
    'privacy.setting.gemini.telemetry.enabled.description': '禁用 Gemini CLI OpenTelemetry 上报。',
    'privacy.setting.gemini.telemetry.enabled.impact': '阻止遥测日志、指标和追踪被导出。',
    'privacy.setting.gemini.telemetry.traces.title': '详细追踪',
    'privacy.setting.gemini.telemetry.traces.description': '禁用详细遥测追踪。',
    'privacy.setting.gemini.telemetry.traces.impact': '避免工具输出和文件读取追踪数据等详细属性。',
    'privacy.setting.gemini.telemetry.logPrompts.title': '提示词日志',
    'privacy.setting.gemini.telemetry.logPrompts.description': '阻止提示词包含在遥测日志中。',
    'privacy.setting.gemini.telemetry.logPrompts.impact': '即使以后启用遥测，也保持提示词文本不进入遥测。',
    'privacy.setting.gemini.general.logRagSnippets.title': 'RAG 片段日志',
    'privacy.setting.gemini.general.logRagSnippets.description': '禁用完整 RAG 片段的本地日志记录。',
    'privacy.setting.gemini.general.logRagSnippets.impact':
      '避免将检索到的代码自定义片段写入本地调试日志。',
    'privacy.setting.gemini.general.checkpointing.enabled.title': '会话检查点',
    'privacy.setting.gemini.general.checkpointing.enabled.description': '禁用会话检查点。',
    'privacy.setting.gemini.general.checkpointing.enabled.impact': '避免额外保存工作状态恢复快照。',
    'privacy.setting.gemini.general.sessionRetention.enabled.title': '会话清理',
    'privacy.setting.gemini.general.sessionRetention.enabled.description': '保持自动会话清理启用。',
    'privacy.setting.gemini.general.sessionRetention.enabled.impact': '确保旧 Gemini CLI 聊天可被自动清理。',
    'privacy.setting.gemini.general.sessionRetention.maxAge.title': '聊天保留窗口',
    'privacy.setting.gemini.general.sessionRetention.maxAge.description': '缩短 Gemini CLI 聊天保留窗口。',
    'privacy.setting.gemini.general.sessionRetention.maxAge.impact':
      '相比默认 30 天保留窗口，保留更少本地聊天记录。',
    'privacy.setting.gemini.tools.sandboxNetworkAccess.title': '沙盒网络访问',
    'privacy.setting.gemini.tools.sandboxNetworkAccess.description': '禁用 Gemini CLI 沙盒内的网络访问。',
    'privacy.setting.gemini.tools.sandboxNetworkAccess.impact':
      '除非之后显式更改，否则让沙盒工具执行保持离线。',
    'privacy.setting.gemini.tools.exclude.web.title': 'Web 工具',
    'privacy.setting.gemini.tools.exclude.web.description': '排除 Gemini CLI Web 搜索和 Web 抓取工具。',
    'privacy.setting.gemini.tools.exclude.web.impact':
      '默认阻止模型使用内置 Web 搜索或 URL 抓取工具。',
    'privacy.setting.gemini.experimental.directWebFetch.title': '直接 Web 抓取',
    'privacy.setting.gemini.experimental.directWebFetch.description': '禁用直接 Web 抓取行为。',
    'privacy.setting.gemini.experimental.directWebFetch.impact': '避免绕过 LLM 摘要的 Web 抓取路径。',
    'privacy.setting.gemini.advanced.ignoreLocalEnv.title': '本地 .env 加载',
    'privacy.setting.gemini.advanced.ignoreLocalEnv.description': '忽略通用项目 .env 文件。',
    'privacy.setting.gemini.advanced.ignoreLocalEnv.impact': '减少项目密钥被意外加载到 Gemini CLI 进程。',
    'privacy.setting.gemini.security.environmentVariableRedaction.enabled.title': '环境变量脱敏',
    'privacy.setting.gemini.security.environmentVariableRedaction.enabled.description': '启用敏感环境变量脱敏。',
    'privacy.setting.gemini.security.environmentVariableRedaction.enabled.impact':
      '对可能包含密钥的环境变量进行脱敏。',
    'privacy.setting.gemini.security.disableYoloMode.title': 'YOLO 模式',
    'privacy.setting.gemini.security.disableYoloMode.description': '即使通过标志请求，也禁用 YOLO 模式。',
    'privacy.setting.gemini.security.disableYoloMode.impact': '防止意外启用广泛自动审批。',
    'privacy.setting.gemini.security.disableAlwaysAllow.title': '始终允许',
    'privacy.setting.gemini.security.disableAlwaysAllow.description': '禁用持久化“始终允许”选择。',
    'privacy.setting.gemini.security.disableAlwaysAllow.impact': '减少可能泄漏到未来会话的长期工具审批。',
    'privacy.setting.gemini.security.enablePermanentToolApproval.title': '永久工具审批',
    'privacy.setting.gemini.security.enablePermanentToolApproval.description': '禁用永久工具审批。',
    'privacy.setting.gemini.security.enablePermanentToolApproval.impact':
      '避免从确认对话框添加未来会话审批。',
    'privacy.setting.gemini.security.blockGitExtensions.title': 'Git 扩展',
    'privacy.setting.gemini.security.blockGitExtensions.description': '阻止安装和加载来自 Git 的扩展。',
    'privacy.setting.gemini.security.blockGitExtensions.impact':
      '减少对远程扩展代码及扩展提供工具的暴露。',
    'privacy.setting.gemini.agents.browser.confirmSensitiveActions.title': '敏感浏览器操作',
    'privacy.setting.gemini.agents.browser.confirmSensitiveActions.description': '对敏感浏览器操作要求确认。',
    'privacy.setting.gemini.agents.browser.confirmSensitiveActions.impact':
      '在填写表单或运行浏览器脚本前要求手动确认。',
    'privacy.setting.gemini.agents.browser.blockFileUploads.title': '浏览器文件上传',
    'privacy.setting.gemini.agents.browser.blockFileUploads.description': '阻止浏览器 Agent 上传文件。',
    'privacy.setting.gemini.agents.browser.blockFileUploads.impact': '防止浏览器自动化上传本地文件。',
    'privacy.setting.gemini.experimental.voiceMode.title': '语音模式',
    'privacy.setting.gemini.experimental.voiceMode.description': '禁用实验性语音模式。',
    'privacy.setting.gemini.experimental.voiceMode.impact':
      '避免可能将录音发送到云端转录后端的语音工作流。',
    'privacy.setting.gemini.experimental.autoMemory.title': '自动记忆',
    'privacy.setting.gemini.experimental.autoMemory.description': '禁用从过去会话自动提取记忆和技能。',
    'privacy.setting.gemini.experimental.autoMemory.impact':
      '防止后台模型调用处理选定的本地转录内容。',
    'privacy.setting.gemini.context.loadMemoryFromIncludeDirectories.title': 'include 目录记忆',
    'privacy.setting.gemini.context.loadMemoryFromIncludeDirectories.description':
      '禁用从 include 目录加载记忆文件。',
    'privacy.setting.gemini.context.loadMemoryFromIncludeDirectories.impact':
      '默认将 /memory reload 限定在当前目录。',
    'privacy.setting.gemini.skills.enabled.title': 'Agent 技能',
    'privacy.setting.gemini.skills.enabled.description': '禁用 Gemini CLI Agent 技能。',
    'privacy.setting.gemini.skills.enabled.impact': '避免将本地技能指令注入后续 Agent 上下文。'
  }
})

type PrivacyMessageKey = Parameters<typeof t>[0]
type SettingTextField = 'title' | 'description' | 'impact'

const loading = ref(true)
const savingAll = ref(false)
const savingId = ref('')
const selectedTarget = ref<PrivacyTarget>('codex')
const privacyStatus = ref<PrivacyConfigStatus | null>(null)
const lastApply = ref<PrivacyConfigApplyResult | null>(null)
const translate = t as (key: string, params?: Record<string, string>) => string
const {
  syncEdits,
  editFor,
  markEditSet,
  useStrict,
  unsetEdit,
  resetEdit,
  isEditChanged,
  changeForSetting,
  canEdit
} = useAgentPrivacyEditor()
let loadRequestId = 0
let saveRequestId = 0

const groupMessageKeys: Record<string, PrivacyMessageKey> = {
  Telemetry: 'privacy.settingGroup.telemetry',
  Network: 'privacy.settingGroup.network',
  'Local history': 'privacy.settingGroup.localHistory',
  Memory: 'privacy.settingGroup.memory',
  Environment: 'privacy.settingGroup.environment',
  Usage: 'privacy.settingGroup.usage',
  'Local retention': 'privacy.settingGroup.localRetention',
  Approval: 'privacy.settingGroup.approval',
  Extensions: 'privacy.settingGroup.extensions',
  Browser: 'privacy.settingGroup.browser',
  Voice: 'privacy.settingGroup.voice'
}
const settingMessageBases: Partial<Record<PrivacyTarget, Record<string, string>>> = {
  codex: {
    'analytics.enabled': 'privacy.setting.codex.analytics.enabled',
    'otel.exporter': 'privacy.setting.codex.otel.exporter',
    'otel.trace_exporter': 'privacy.setting.codex.otel.trace_exporter',
    'otel.metrics_exporter': 'privacy.setting.codex.otel.metrics_exporter',
    'otel.log_user_prompt': 'privacy.setting.codex.otel.log_user_prompt',
    web_search: 'privacy.setting.codex.web_search',
    'history.persistence': 'privacy.setting.codex.history.persistence',
    'features.memories': 'privacy.setting.codex.features.memories',
    'memories.generate_memories': 'privacy.setting.codex.memories.generate_memories',
    'memories.use_memories': 'privacy.setting.codex.memories.use_memories',
    'memories.disable_on_external_context': 'privacy.setting.codex.memories.disable_on_external_context',
    'sandbox_workspace_write.network_access': 'privacy.setting.codex.sandbox_workspace_write.network_access',
    'shell_environment_policy.inherit': 'privacy.setting.codex.shell_environment_policy.inherit',
    'shell_environment_policy.ignore_default_excludes': 'privacy.setting.codex.shell_environment_policy.ignore_default_excludes'
  },
  gemini: {
    'privacy.usageStatisticsEnabled': 'privacy.setting.gemini.privacy.usageStatisticsEnabled',
    'telemetry.enabled': 'privacy.setting.gemini.telemetry.enabled',
    'telemetry.traces': 'privacy.setting.gemini.telemetry.traces',
    'telemetry.logPrompts': 'privacy.setting.gemini.telemetry.logPrompts',
    'general.logRagSnippets': 'privacy.setting.gemini.general.logRagSnippets',
    'general.checkpointing.enabled': 'privacy.setting.gemini.general.checkpointing.enabled',
    'general.sessionRetention.enabled': 'privacy.setting.gemini.general.sessionRetention.enabled',
    'general.sessionRetention.maxAge': 'privacy.setting.gemini.general.sessionRetention.maxAge',
    'tools.sandboxNetworkAccess': 'privacy.setting.gemini.tools.sandboxNetworkAccess',
    'tools.exclude.web': 'privacy.setting.gemini.tools.exclude.web',
    'experimental.directWebFetch': 'privacy.setting.gemini.experimental.directWebFetch',
    'advanced.ignoreLocalEnv': 'privacy.setting.gemini.advanced.ignoreLocalEnv',
    'security.environmentVariableRedaction.enabled':
      'privacy.setting.gemini.security.environmentVariableRedaction.enabled',
    'security.disableYoloMode': 'privacy.setting.gemini.security.disableYoloMode',
    'security.disableAlwaysAllow': 'privacy.setting.gemini.security.disableAlwaysAllow',
    'security.enablePermanentToolApproval': 'privacy.setting.gemini.security.enablePermanentToolApproval',
    'security.blockGitExtensions': 'privacy.setting.gemini.security.blockGitExtensions',
    'agents.browser.confirmSensitiveActions': 'privacy.setting.gemini.agents.browser.confirmSensitiveActions',
    'agents.browser.blockFileUploads': 'privacy.setting.gemini.agents.browser.blockFileUploads',
    'experimental.voiceMode': 'privacy.setting.gemini.experimental.voiceMode',
    'experimental.autoMemory': 'privacy.setting.gemini.experimental.autoMemory',
    'context.loadMemoryFromIncludeDirectories':
      'privacy.setting.gemini.context.loadMemoryFromIncludeDirectories',
    'skills.enabled': 'privacy.setting.gemini.skills.enabled'
  }
}

const targetOptions = computed<{ label: string; value: PrivacyTarget }[]>(() => [
  { label: t('privacy.target.codex'), value: 'codex' },
  { label: t('privacy.target.gemini'), value: 'gemini' },
  { label: t('privacy.target.claude'), value: 'claude' },
  { label: t('privacy.target.codebuddy'), value: 'codebuddy' }
])
const targetLabel = computed(() => {
  if (privacyStatus.value?.name) return privacyStatus.value.name
  return targetOptions.value.find((option) => option.value === selectedTarget.value)?.label || selectedTarget.value
})
const targetFile = computed(() => (selectedTarget.value === 'codex' ? 'config.toml' : 'settings.json'))
const summary = computed(
  () =>
    privacyStatus.value?.summary || {
      score: 0,
      total: 0,
      hardened: 0,
      attention: 0,
      implicit: 0
    }
)
const settings = computed(() => privacyStatus.value?.settings || [])
const changedSettings = computed(() => settings.value.filter((setting) => canEdit(setting) && isEditChanged(setting)))
const kickerText = computed(() => t('privacy.kicker', { target: targetLabel.value, file: targetFile.value }))
const statusState = computed(() => {
  if (!privacyStatus.value) return { color: 'default', label: t('privacy.status.noStatus') }
  if (metricCounts.value.missingRequired > 0) {
    return { color: 'warning', label: t('privacy.status.needsChange') }
  }
  return { color: 'success', label: t('privacy.status.ready') }
})
const warningList = computed(() => {
  const values = [...(privacyStatus.value?.warnings || []), ...(lastApply.value?.warnings || [])]
  return [...new Set(values.filter(Boolean))]
})
const metricCounts = computed(() => {
  const total = settings.value.length || summary.value.total
  const strictConfigured = settings.value.filter(
    (setting) =>
      setting.configured && privacyValuesEqual(setting.currentValue, strictPrivacyValue(setting), privacyValueType(setting))
  ).length
  const defaultSafe = settings.value.filter((setting) => !setting.configured && setting.status === 'implicit').length
  const customConfigured = settings.value.filter(
    (setting) =>
      setting.configured && !privacyValuesEqual(setting.currentValue, strictPrivacyValue(setting), privacyValueType(setting))
  ).length
  const missingRequired = settings.value.filter((setting) => !setting.configured && setting.status === 'attention').length
  const unsavedChanges = changedSettings.value.length
  return { total, strictConfigured, defaultSafe, customConfigured, missingRequired, unsavedChanges }
})
const groupedSettings = computed(() => {
  const groups = new Map<string, PrivacyConfigSetting[]>()
  for (const setting of settings.value) {
    const group = localizedSettingGroup(setting)
    groups.set(group, [...(groups.get(group) || []), setting])
  }
  return [...groups.entries()].map(([name, items]) => ({ name, items }))
})

function activeSettingTarget(): PrivacyTarget | undefined {
  const target = privacyStatus.value?.target || selectedTarget.value
  if (target === 'codex' || target === 'gemini') return target
  return undefined
}

function localizedMessage(key: PrivacyMessageKey | undefined, fallback: string) {
  if (!key || locale.value === 'en') return fallback
  const localized = t(key)
  return localized === key ? fallback : localized
}

function localizedSettingGroup(setting: PrivacyConfigSetting) {
  if (!setting.group) return t('privacy.group.default')
  return localizedMessage(groupMessageKeys[setting.group], setting.group)
}

function settingMessageKey(setting: PrivacyConfigSetting, field: SettingTextField): PrivacyMessageKey | undefined {
  const target = activeSettingTarget()
  const base = target ? settingMessageBases[target]?.[setting.id] : undefined
  if (!base) return undefined
  return `${base}.${field}` as PrivacyMessageKey
}

function localizedSettingText(setting: PrivacyConfigSetting, field: SettingTextField, fallback: string) {
  return localizedMessage(settingMessageKey(setting, field), fallback)
}

function localizedSettingTitle(setting: PrivacyConfigSetting) {
  return localizedSettingText(setting, 'title', setting.title)
}

function localizedSettingDescription(setting: PrivacyConfigSetting) {
  return localizedSettingText(setting, 'description', setting.description)
}

function localizedSettingImpact(setting: PrivacyConfigSetting) {
  return localizedSettingText(setting, 'impact', setting.impact)
}

function formatConfigValue(value: unknown) {
  return formatPrivacyConfigValue(value, t('privacy.value.unset'))
}

function settingState(setting: PrivacyConfigSetting) {
  if (setting.configured && privacyValuesEqual(setting.currentValue, strictPrivacyValue(setting), privacyValueType(setting))) {
    return { color: 'success', label: t('privacy.status.hardened') }
  }
  if (setting.configured) return { color: 'processing', label: t('privacy.meta.customConfigured') }
  if (setting.status === 'implicit') return { color: 'default', label: t('privacy.status.implicit') }
  return { color: 'warning', label: t('privacy.value.notConfigured') }
}

function settingCardClass(setting: PrivacyConfigSetting) {
  return {
    'privacy-setting-card': true,
    'is-attention': !setting.configured && setting.status === 'attention',
    'is-changed': isEditChanged(setting)
  }
}

async function load() {
  const requestId = ++loadRequestId
  loading.value = true
  try {
    const status = await api.getAgentPrivacy(selectedTarget.value)
    if (requestId !== loadRequestId) return
    privacyStatus.value = status
    syncEdits(status)
  } catch (error) {
    if (requestId !== loadRequestId) return
    message.error(error instanceof Error ? error.message : t('privacy.message.loadFailed'))
  } finally {
    if (requestId === loadRequestId) loading.value = false
  }
}

async function saveSettings(records: PrivacyConfigSetting[], saveAll = false) {
  const changes = records.filter((setting) => canEdit(setting) && isEditChanged(setting)).map(changeForSetting)
  if (!changes.length) {
    message.info(t('privacy.message.noChanges'))
    return
  }

  const requestId = ++saveRequestId
  const target = selectedTarget.value
  if (saveAll) savingAll.value = true
  else savingId.value = changes[0].id

  try {
    const result = await api.applyAgentPrivacyChanges(target, changes)
    if (requestId !== saveRequestId || selectedTarget.value !== target) return
    if (result.status.target !== target) {
      message.error(t('privacy.message.targetMismatch'))
      return
    }
    privacyStatus.value = result.status
    lastApply.value = result
    syncEdits(result.status)
    if (result.changed?.length) {
      message.success(t('privacy.message.saved', { count: formatNumber(result.changed.length) }))
    } else {
      message.info(t('privacy.message.noChanges'))
    }
  } catch (error) {
    if (requestId !== saveRequestId || selectedTarget.value !== target) return
    message.error(error instanceof Error ? error.message : t('privacy.message.saveFailed'))
  } finally {
    if (requestId === saveRequestId) {
      savingAll.value = false
      savingId.value = ''
    }
  }
}

onMounted(load)
watch(selectedTarget, () => {
  saveRequestId++
  savingAll.value = false
  savingId.value = ''
  lastApply.value = null
  load()
})
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <PrivacySummaryPanel
        v-model:selected-target="selectedTarget"
        :t="translate"
        :target-options="targetOptions"
        :kicker-text="kickerText"
        :status-state="statusState"
        :privacy-status="privacyStatus"
        :last-apply="lastApply"
        :metric-counts="metricCounts"
        :changed-count="changedSettings.length"
        :saving-all="savingAll"
        :saving-id="savingId"
        :warning-list="warningList"
        :target-label="targetLabel"
        :format-number="formatNumber"
        :format-config-value="formatConfigValue"
        @refresh="load"
        @save-all="saveSettings(changedSettings, true)"
      />

      <PrivacySettingsPanel
        :t="translate"
        :settings="settings"
        :grouped-settings="groupedSettings"
        :saving-all="savingAll"
        :saving-id="savingId"
        :format-number="formatNumber"
        :format-config-value="formatConfigValue"
        :edit-for="editFor"
        :can-edit="canEdit"
        :is-edit-changed="isEditChanged"
        :setting-state="settingState"
        :setting-card-class="settingCardClass"
        :localized-setting-title="localizedSettingTitle"
        :localized-setting-description="localizedSettingDescription"
        :localized-setting-impact="localizedSettingImpact"
        :value-type="privacyValueType"
        :strict-value="strictPrivacyValue"
        :value-type-label="privacyValueTypeLabel"
        @mark-set="markEditSet"
        @use-strict="useStrict"
        @unset="unsetEdit"
        @reset="resetEdit"
        @save="(setting) => saveSettings([setting])"
      />
    </div>
  </a-spin>
</template>
