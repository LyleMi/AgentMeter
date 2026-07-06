package privacy

import (
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type CodeBuddyAdapter struct {
	Now        func() time.Time
	ConfigPath string
}

func NewCodeBuddyAdapter() CodeBuddyAdapter {
	return CodeBuddyAdapter{Now: time.Now}
}

func (a CodeBuddyAdapter) Status() (model.PrivacyConfigStatus, error) {
	return a.jsonAdapter().status()
}

func (a CodeBuddyAdapter) Apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().apply(settingIDs)
}

func (a CodeBuddyAdapter) ApplyChanges(edits []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().applyChanges(edits)
}

func (a CodeBuddyAdapter) ApplyProfile(profile string) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().applyProfile(profile)
}

func (a CodeBuddyAdapter) jsonAdapter() jsonPrivacyAdapter {
	return codeBuddyJSONAdapter(codeBuddyJSONSpec.settingsPathFunc(a.ConfigPath), a.Now)
}

func codeBuddyJSONAdapter(settingsPath func() (string, error), now func() time.Time) jsonPrivacyAdapter {
	return codeBuddyJSONSpec.adapter(settingsPath, now)
}

func codeBuddySettingsPath() (string, error) {
	return codeBuddyJSONSpec.settingsPath("")
}

func buildCodeBuddyStatus(path string, exists bool, content []byte, warnings []string) model.PrivacyConfigStatus {
	return codeBuddyJSONSpec.buildStatus(privacyConfigFile{path: path, exists: exists, content: content, warnings: warnings})
}

var codeBuddyJSONSpec = jsonAdapterSpec{
	target:      "codebuddy",
	name:        "CodeBuddy Code/IDE",
	agentName:   "CodeBuddy",
	definitions: codeBuddySettingDefinitions,
	path: jsonSettingsPathSpec{
		overrideEnv:  "AGENTMETER_CODEBUDDY_SETTINGS_PATH",
		configDirEnv: "CODEBUDDY_CONFIG_DIR",
		homeDirName:  ".codebuddy",
	},
}

var codeBuddySettingDefinitions = []jsonSettingDefinition{
	{
		ID:          "env.DISABLE_TELEMETRY",
		Group:       "Telemetry",
		Title:       "Telemetry",
		Description: "Disables CodeBuddy Code OpenTelemetry export through the user settings environment.",
		Key:         "env.DISABLE_TELEMETRY",
		Desired:     "1",
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Disables all CodeBuddy telemetry with the documented highest-priority opt-out.",
	},
	{
		ID:          "env.CODEBUDDY_CODE_ENABLE_TELEMETRY",
		Group:       "Telemetry",
		Title:       "Telemetry opt-in",
		Description: "Prevents the CodeBuddy telemetry opt-in environment variable from enabling export later.",
		Key:         "env.CODEBUDDY_CODE_ENABLE_TELEMETRY",
		Desired:     "0",
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Keeps user-level settings from explicitly opting into trace export.",
	},
	{
		ID:          "env.CLAUDE_CODE_ENABLE_TELEMETRY",
		Group:       "Telemetry",
		Title:       "Claude telemetry alias",
		Description: "Prevents the Claude-compatible telemetry opt-in variable from enabling CodeBuddy export.",
		Key:         "env.CLAUDE_CODE_ENABLE_TELEMETRY",
		Desired:     "0",
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Covers CodeBuddy's Claude Code telemetry compatibility environment variable.",
	},
	{
		ID:          "env.OTEL_TRACES_EXPORTER",
		Group:       "Telemetry",
		Title:       "OTel exporter",
		Description: "Disables the OpenTelemetry trace exporter in user settings.",
		Key:         "env.OTEL_TRACES_EXPORTER",
		Desired:     "none",
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Prevents user settings from sending CodeBuddy traces to an external collector.",
	},
	{
		ID:          "env.OTEL_LOG_USER_PROMPTS",
		Group:       "Telemetry",
		Title:       "Prompt recording",
		Description: "Keeps user prompt text out of OpenTelemetry spans.",
		Key:         "env.OTEL_LOG_USER_PROMPTS",
		Desired:     "0",
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Avoids recording full user prompts if telemetry is enabled elsewhere.",
	},
	{
		ID:          "env.OTEL_LOG_TOOL_DETAILS",
		Group:       "Telemetry",
		Title:       "Tool detail recording",
		Description: "Keeps tool parameters out of OpenTelemetry spans.",
		Key:         "env.OTEL_LOG_TOOL_DETAILS",
		Desired:     "0",
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Avoids recording file paths, commands, URLs, searches, and tool input attributes.",
	},
	{
		ID:          "env.OTEL_LOG_TOOL_CONTENT",
		Group:       "Telemetry",
		Title:       "Tool content recording",
		Description: "Keeps full tool input and output out of OpenTelemetry span events.",
		Key:         "env.OTEL_LOG_TOOL_CONTENT",
		Desired:     "0",
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Avoids recording tool inputs and results that can include source code or secrets.",
	},
	{
		ID:          "env.OTEL_LOG_RAW_API_BODIES",
		Group:       "Telemetry",
		Title:       "Raw API body recording",
		Description: "Keeps raw API request and response body recording disabled.",
		Key:         "env.OTEL_LOG_RAW_API_BODIES",
		Desired:     "0",
		DefaultSafe: true,
		Recommended: true,
		Impact:      "Avoids future full request/response body capture if the reserved switch becomes active.",
	},
	{
		ID:          "env.DISABLE_ERROR_REPORTING",
		Group:       "Reporting",
		Title:       "Error reporting",
		Description: "Disables CodeBuddy error reporting through the user settings environment.",
		Key:         "env.DISABLE_ERROR_REPORTING",
		Desired:     "1",
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Reduces diagnostic error payloads sent outside the machine.",
	},
	{
		ID:          "env.DISABLE_FEEDBACK_COMMAND",
		Group:       "Reporting",
		Title:       "Feedback command",
		Description: "Disables the CodeBuddy feedback command.",
		Key:         "env.DISABLE_FEEDBACK_COMMAND",
		Desired:     "1",
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Prevents feedback command submissions from this environment.",
	},
	{
		ID:          "env.DISABLE_AUTOUPDATER",
		Group:       "Network",
		Title:       "Auto updater",
		Description: "Disables CodeBuddy automatic update checks through the user settings environment.",
		Key:         "env.DISABLE_AUTOUPDATER",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Avoids background update traffic controlled by this settings file.",
	},
	{
		ID:          "autoUpdates",
		Group:       "Network",
		Title:       "Auto updates",
		Description: "Keeps CodeBuddy auto-update settings disabled.",
		Key:         "autoUpdates",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents this user settings file from enabling automatic update checks.",
	},
	{
		ID:          "cleanupPeriodDays",
		Group:       "Local retention",
		Title:       "Chat retention window",
		Description: "Reduces the CodeBuddy local chat history retention window.",
		Key:         "cleanupPeriodDays",
		Desired:     7,
		DefaultSafe: false,
		Impact:      "Keeps fewer local chat records than CodeBuddy's documented 30 day default.",
	},
	{
		ID:          "memory.autoMemoryEnabled",
		Group:       "Memory",
		Title:       "Auto memory",
		Description: "Disables CodeBuddy automatic memory storage.",
		Key:         "memory.autoMemoryEnabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Prevents autonomous persistent memory extraction from conversations.",
	},
	{
		ID:          "env.CODEBUDDY_DISABLE_AUTO_MEMORY",
		Group:       "Memory",
		Title:       "Auto memory environment",
		Description: "Disables CodeBuddy automatic memory with the documented environment variable.",
		Key:         "env.CODEBUDDY_DISABLE_AUTO_MEMORY",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Provides an environment-level fallback for disabling auto memory.",
	},
	{
		ID:          "memory.memoryExtraction",
		Group:       "Memory",
		Title:       "Memory extraction",
		Description: "Disables background memory extraction.",
		Key:         "memory.memoryExtraction",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids extracting durable memory from conversations at the end of a session.",
	},
	{
		ID:          "memory.teamMemory.enabled",
		Group:       "Memory",
		Title:       "Team memory",
		Description: "Disables team memory mode from user settings.",
		Key:         "memory.teamMemory.enabled",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents writing project memories into shared team memory storage from this setting.",
	},
	{
		ID:          "includeCoAuthoredBy",
		Group:       "Attribution",
		Title:       "Commit attribution",
		Description: "Disables CodeBuddy co-authored-by attribution in commits and pull requests.",
		Key:         "includeCoAuthoredBy",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Avoids adding CodeBuddy attribution metadata to generated git content.",
	},
	{
		ID:          "trustAll",
		Group:       "Trust",
		Title:       "Trust all directories",
		Description: "Keeps CodeBuddy directory trust prompts enabled.",
		Key:         "trustAll",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents this settings file from globally trusting every working directory.",
	},
	{
		ID:          "permissions.defaultMode",
		Group:       "Permissions",
		Title:       "Default permission mode",
		Description: "Keeps CodeBuddy's default permission mode at the normal reviewed mode.",
		Key:         "permissions.defaultMode",
		Desired:     "default",
		DefaultSafe: true,
		Impact:      "Avoids starting sessions in auto, dontAsk, or bypassPermissions mode.",
	},
	{
		ID:          "permissions.disableBypassPermissionsMode",
		Group:       "Permissions",
		Title:       "Bypass permission mode",
		Description: "Disables activation of CodeBuddy bypassPermissions mode.",
		Key:         "permissions.disableBypassPermissionsMode",
		Desired:     "disable",
		DefaultSafe: false,
		Impact:      "Disables the dangerous skip-permissions mode and related CLI flags.",
	},
	{
		ID:          "permissions.disableAutoMode",
		Group:       "Permissions",
		Title:       "Auto permission mode",
		Description: "Disables activation of CodeBuddy auto permission mode.",
		Key:         "permissions.disableAutoMode",
		Desired:     "disable",
		DefaultSafe: false,
		Impact:      "Prevents automatic permission-mode decisions from this user settings file.",
	},
	{
		ID:          "permissions.deny",
		Group:       "Permissions",
		Title:       "Sensitive and web deny rules",
		Description: "Adds CodeBuddy deny rules for web access, common download commands, and sensitive files.",
		Key:         "permissions.deny",
		Desired: []string{
			"WebFetch",
			"WebSearch",
			"Bash(curl:*)",
			"Bash(wget:*)",
			"Read(./.env)",
			"Read(./.env.*)",
			"Read(./secrets/**)",
			"Read(~/.ssh/**)",
			"Read(~/.aws/**)",
			"Edit(**/*.env)",
			"Edit(**/*.key)",
			"Edit(**/*.pem)",
		},
		DefaultSafe: false,
		Impact:      "Reduces accidental exposure through web tools, download commands, and common secret-bearing files.",
		MergeArray:  true,
	},
	{
		ID:          "disableAllHooks",
		Group:       "Hooks",
		Title:       "Hooks",
		Description: "Disables CodeBuddy hooks from this user settings file.",
		Key:         "disableAllHooks",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Prevents hook commands from running before or after tool execution.",
	},
	{
		ID:          "allowUntrustedFrontmatterHooks",
		Group:       "Hooks",
		Title:       "Untrusted frontmatter hooks",
		Description: "Keeps untrusted agent and skill frontmatter hooks disabled.",
		Key:         "allowUntrustedFrontmatterHooks",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents local or marketplace markdown files from silently launching shell commands.",
	},
	{
		ID:          "enableAllProjectMcpServers",
		Group:       "MCP",
		Title:       "Project MCP auto-approval",
		Description: "Disables automatic approval of all project MCP servers.",
		Key:         "enableAllProjectMcpServers",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents project .mcp.json files from enabling every MCP server without review.",
	},
	{
		ID:          "enabledMcpjsonServers",
		Group:       "MCP",
		Title:       "Approved project MCP servers",
		Description: "Clears explicitly approved project MCP server names from user settings.",
		Key:         "enabledMcpjsonServers",
		Desired:     []string{},
		DefaultSafe: true,
		Impact:      "Requires project MCP servers to be reviewed instead of pre-approved.",
	},
}
