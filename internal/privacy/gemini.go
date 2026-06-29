package privacy

import (
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type GeminiAdapter struct {
	Now        func() time.Time
	ConfigPath string
}

func NewGeminiAdapter() GeminiAdapter {
	return GeminiAdapter{Now: time.Now}
}

func (a GeminiAdapter) Status() (model.PrivacyConfigStatus, error) {
	return a.jsonAdapter().status()
}

func (a GeminiAdapter) Apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().apply(settingIDs)
}

func (a GeminiAdapter) ApplyChanges(edits []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().applyChanges(edits)
}

func (a GeminiAdapter) ApplyProfile(profile string) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().applyProfile(profile)
}

func (a GeminiAdapter) jsonAdapter() jsonPrivacyAdapter {
	return geminiJSONAdapter(geminiJSONSpec.settingsPathFunc(a.ConfigPath), a.Now)
}

func geminiJSONAdapter(settingsPath func() (string, error), now func() time.Time) jsonPrivacyAdapter {
	return geminiJSONSpec.adapter(settingsPath, now)
}

func geminiSettingsPath() (string, error) {
	return geminiJSONSpec.settingsPath("")
}

func buildGeminiStatus(path string, exists bool, content []byte, warnings []string) model.PrivacyConfigStatus {
	return geminiJSONSpec.buildStatus(path, exists, content, warnings)
}

var geminiJSONSpec = jsonAdapterSpec{
	target:      "gemini",
	name:        "Gemini CLI",
	agentName:   "Gemini",
	definitions: geminiSettingDefinitions,
	path: jsonSettingsPathSpec{
		overrideEnv: "AGENTMETER_GEMINI_SETTINGS_PATH",
		homeDirName: ".gemini",
	},
}

var geminiSettingDefinitions = []jsonSettingDefinition{
	{
		ID:          "privacy.usageStatisticsEnabled",
		Group:       "Usage",
		Title:       "Usage statistics",
		Description: "Opts out of Gemini CLI usage statistics collection.",
		Key:         "privacy.usageStatisticsEnabled",
		Desired:     false,
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Prevents Gemini CLI usage statistics from being sent to Google.",
	},
	{
		ID:          "telemetry.enabled",
		Group:       "Telemetry",
		Title:       "OpenTelemetry",
		Description: "Disables Gemini CLI OpenTelemetry emission.",
		Key:         "telemetry.enabled",
		Desired:     false,
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Prevents telemetry logs, metrics, and traces from being exported.",
	},
	{
		ID:          "telemetry.traces",
		Group:       "Telemetry",
		Title:       "Detailed traces",
		Description: "Disables detailed telemetry traces.",
		Key:         "telemetry.traces",
		Desired:     false,
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Avoids detailed attributes such as tool output and file-read trace data.",
	},
	{
		ID:          "telemetry.logPrompts",
		Group:       "Telemetry",
		Title:       "Prompt logging",
		Description: "Prevents prompts from being included in telemetry logs.",
		Key:         "telemetry.logPrompts",
		Desired:     false,
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Keeps prompt text out of telemetry if telemetry is enabled later.",
	},
	{
		ID:          "general.logRagSnippets",
		Group:       "Telemetry",
		Title:       "RAG snippet logging",
		Description: "Disables local logging of full RAG snippets.",
		Key:         "general.logRagSnippets",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids writing retrieved code customization snippets to local debug logs.",
	},
	{
		ID:          "general.checkpointing.enabled",
		Group:       "Local retention",
		Title:       "Session checkpointing",
		Description: "Disables session checkpointing.",
		Key:         "general.checkpointing.enabled",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids extra recovery snapshots of working state.",
	},
	{
		ID:          "general.sessionRetention.enabled",
		Group:       "Local retention",
		Title:       "Session cleanup",
		Description: "Keeps automatic session cleanup enabled.",
		Key:         "general.sessionRetention.enabled",
		Desired:     true,
		DefaultSafe: true,
		Impact:      "Ensures old Gemini CLI chats are eligible for automatic cleanup.",
	},
	{
		ID:          "general.sessionRetention.maxAge",
		Group:       "Local retention",
		Title:       "Chat retention window",
		Description: "Reduces the Gemini CLI chat retention window.",
		Key:         "general.sessionRetention.maxAge",
		Desired:     "7d",
		DefaultSafe: false,
		Impact:      "Keeps fewer local chat records than the default 30 day retention window.",
	},
	{
		ID:          "tools.sandboxNetworkAccess",
		Group:       "Network",
		Title:       "Sandbox network access",
		Description: "Disables network access inside the Gemini CLI sandbox.",
		Key:         "tools.sandboxNetworkAccess",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Keeps sandboxed tool execution offline unless explicitly changed later.",
	},
	{
		ID:          "tools.exclude.web",
		Group:       "Network",
		Title:       "Web tools",
		Description: "Excludes Gemini CLI web search and web fetch tools.",
		Key:         "tools.exclude",
		Desired:     []string{"google_web_search", "web_fetch"},
		DefaultSafe: false,
		Impact:      "Prevents the model from using built-in web search or URL fetch tools by default.",
		MergeArray:  true,
	},
	{
		ID:          "experimental.directWebFetch",
		Group:       "Network",
		Title:       "Direct web fetch",
		Description: "Disables direct web fetch behavior.",
		Key:         "experimental.directWebFetch",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids web fetch paths that bypass LLM summarization.",
	},
	{
		ID:          "advanced.ignoreLocalEnv",
		Group:       "Environment",
		Title:       "Local .env loading",
		Description: "Ignores generic project .env files.",
		Key:         "advanced.ignoreLocalEnv",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Reduces accidental loading of project secrets into the Gemini CLI process.",
	},
	{
		ID:          "security.environmentVariableRedaction.enabled",
		Group:       "Environment",
		Title:       "Environment variable redaction",
		Description: "Enables redaction for sensitive environment variables.",
		Key:         "security.environmentVariableRedaction.enabled",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Redacts environment variables that may contain secrets.",
	},
	{
		ID:          "security.disableYoloMode",
		Group:       "Approval",
		Title:       "YOLO mode",
		Description: "Disables YOLO mode even when requested by flag.",
		Key:         "security.disableYoloMode",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Prevents broad automatic approval from being enabled accidentally.",
	},
	{
		ID:          "security.disableAlwaysAllow",
		Group:       "Approval",
		Title:       "Always allow",
		Description: "Disables persistent Always allow choices.",
		Key:         "security.disableAlwaysAllow",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Reduces long-lived tool approvals that can leak into future sessions.",
	},
	{
		ID:          "security.enablePermanentToolApproval",
		Group:       "Approval",
		Title:       "Permanent tool approval",
		Description: "Disables permanent tool approval.",
		Key:         "security.enablePermanentToolApproval",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids future-session approvals being added from confirmation dialogs.",
	},
	{
		ID:          "security.blockGitExtensions",
		Group:       "Extensions",
		Title:       "Git extensions",
		Description: "Blocks installing and loading extensions from Git.",
		Key:         "security.blockGitExtensions",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Reduces exposure to remote extension code and extension-provided tools.",
	},
	{
		ID:          "agents.browser.confirmSensitiveActions",
		Group:       "Browser",
		Title:       "Sensitive browser actions",
		Description: "Requires confirmation for sensitive browser actions.",
		Key:         "agents.browser.confirmSensitiveActions",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Requires manual confirmation before filling forms or running browser scripts.",
	},
	{
		ID:          "agents.browser.blockFileUploads",
		Group:       "Browser",
		Title:       "Browser file uploads",
		Description: "Blocks file uploads from the browser agent.",
		Key:         "agents.browser.blockFileUploads",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Prevents browser automation from uploading local files.",
	},
	{
		ID:          "experimental.voiceMode",
		Group:       "Voice",
		Title:       "Voice mode",
		Description: "Disables experimental voice mode.",
		Key:         "experimental.voiceMode",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids voice workflows that may send recordings to a cloud transcription backend.",
	},
	{
		ID:          "experimental.autoMemory",
		Group:       "Memory",
		Title:       "Auto Memory",
		Description: "Disables automatic memory and skill extraction from past sessions.",
		Key:         "experimental.autoMemory",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents background model calls over selected local transcript content.",
	},
	{
		ID:          "context.loadMemoryFromIncludeDirectories",
		Group:       "Memory",
		Title:       "Include-directory memory",
		Description: "Disables loading memory files from include directories.",
		Key:         "context.loadMemoryFromIncludeDirectories",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Keeps /memory reload scoped to the current directory by default.",
	},
	{
		ID:          "skills.enabled",
		Group:       "Memory",
		Title:       "Agent skills",
		Description: "Disables Gemini CLI agent skills.",
		Key:         "skills.enabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Avoids injecting local skill instructions into future agent context.",
	},
}
