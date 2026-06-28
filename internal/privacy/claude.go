package privacy

import (
	"time"

	"AgentMeter/internal/model"
)

type ClaudeAdapter struct {
	Now        func() time.Time
	ConfigPath string
}

func NewClaudeAdapter() ClaudeAdapter {
	return ClaudeAdapter{Now: time.Now}
}

func (a ClaudeAdapter) Status() (model.PrivacyConfigStatus, error) {
	return a.jsonAdapter().status()
}

func (a ClaudeAdapter) Apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().apply(settingIDs)
}

func (a ClaudeAdapter) ApplyChanges(edits []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().applyChanges(edits)
}

func (a ClaudeAdapter) ApplyProfile(profile string) (model.PrivacyConfigApplyResult, error) {
	return a.jsonAdapter().applyProfile(profile)
}

func (a ClaudeAdapter) jsonAdapter() jsonPrivacyAdapter {
	return claudeJSONAdapter(a.settingsPath, a.now)
}

func claudeJSONAdapter(settingsPath func() (string, error), now func() time.Time) jsonPrivacyAdapter {
	return jsonPrivacyAdapter{
		target:       "claude",
		name:         "Claude Code",
		agentName:    "Claude Code",
		definitions:  claudeSettingDefinitions,
		settingsPath: settingsPath,
		now:          now,
	}
}

func (a ClaudeAdapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a ClaudeAdapter) settingsPath() (string, error) {
	return jsonSettingsPath(a.ConfigPath, "AGENTMETER_CLAUDE_SETTINGS_PATH", "CLAUDE_CONFIG_DIR", ".claude")
}

func claudeSettingsPath() (string, error) {
	return jsonSettingsPath("", "AGENTMETER_CLAUDE_SETTINGS_PATH", "CLAUDE_CONFIG_DIR", ".claude")
}

func buildClaudeStatus(path string, exists bool, content []byte, warnings []string) model.PrivacyConfigStatus {
	return claudeJSONAdapter(nil, nil).buildStatus(path, exists, content, warnings)
}

var claudeSettingDefinitions = []jsonSettingDefinition{
	{
		ID:          "env.DISABLE_TELEMETRY",
		Group:       "Telemetry",
		Title:       "Telemetry",
		Description: "Disables Claude Code telemetry emission through the user settings environment.",
		Key:         "env.DISABLE_TELEMETRY",
		Desired:     "1",
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Prevents Claude Code telemetry from being enabled through this user settings file.",
	},
	{
		ID:          "env.DISABLE_ERROR_REPORTING",
		Group:       "Telemetry",
		Title:       "Error reporting",
		Description: "Disables Claude Code error reporting through the user settings environment.",
		Key:         "env.DISABLE_ERROR_REPORTING",
		Desired:     "1",
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Reduces diagnostic error payloads sent from Claude Code.",
	},
	{
		ID:          "env.DISABLE_FEEDBACK_COMMAND",
		Group:       "Feedback",
		Title:       "Feedback command",
		Description: "Disables the Claude Code feedback command.",
		Key:         "env.DISABLE_FEEDBACK_COMMAND",
		Desired:     "1",
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Prevents feedback command submissions from this environment.",
	},
	{
		ID:          "env.CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY",
		Group:       "Feedback",
		Title:       "Feedback survey",
		Description: "Disables Claude Code feedback survey prompts.",
		Key:         "env.CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY",
		Desired:     "1",
		DefaultSafe: false,
		Recommended: true,
		Impact:      "Avoids survey flows that may send user feedback outside the machine.",
	},
	{
		ID:          "env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
		Group:       "Network",
		Title:       "Nonessential traffic",
		Description: "Disables nonessential Claude Code network traffic through the user settings environment.",
		Key:         "env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Limits Claude Code to essential service traffic where supported.",
	},
	{
		ID:          "env.CLAUDE_CODE_SUBPROCESS_ENV_SCRUB",
		Group:       "Environment",
		Title:       "Subprocess environment scrub",
		Description: "Requests subprocess environment scrubbing for Claude Code tool execution.",
		Key:         "env.CLAUDE_CODE_SUBPROCESS_ENV_SCRUB",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Reduces accidental exposure of parent-process environment variables to subprocess tools.",
	},
	{
		ID:          "attribution.commit",
		Group:       "Attribution",
		Title:       "Commit attribution",
		Description: "Removes generated commit attribution text.",
		Key:         "attribution.commit",
		Desired:     "",
		DefaultSafe: false,
		Impact:      "Avoids adding Claude attribution metadata to generated commit messages.",
	},
	{
		ID:          "attribution.pr",
		Group:       "Attribution",
		Title:       "Pull request attribution",
		Description: "Removes generated pull request attribution text.",
		Key:         "attribution.pr",
		Desired:     "",
		DefaultSafe: false,
		Impact:      "Avoids adding Claude attribution metadata to generated pull request text.",
	},
	{
		ID:          "attribution.sessionUrl",
		Group:       "Attribution",
		Title:       "Session URL attribution",
		Description: "Disables Claude Code session URL attribution.",
		Key:         "attribution.sessionUrl",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Avoids adding shareable session URLs to generated attribution.",
	},
	{
		ID:          "fileCheckpointingEnabled",
		Group:       "Local retention",
		Title:       "File checkpointing",
		Description: "Disables Claude Code file checkpointing.",
		Key:         "fileCheckpointingEnabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Reduces local recovery snapshots of edited files.",
	},
	{
		ID:          "autoMemoryEnabled",
		Group:       "Memory",
		Title:       "Auto memory",
		Description: "Disables Claude Code automatic memory behavior.",
		Key:         "autoMemoryEnabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Avoids automatic durable memory extraction from conversations.",
	},
	{
		ID:          "disableArtifact",
		Group:       "Artifacts",
		Title:       "Artifacts",
		Description: "Disables Claude Code artifact generation.",
		Key:         "disableArtifact",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Reduces creation of additional generated artifact content outside normal chat output.",
	},
	{
		ID:          "disableClaudeAiConnectors",
		Group:       "Connectors",
		Title:       "Claude AI connectors",
		Description: "Disables Claude AI connector integrations.",
		Key:         "disableClaudeAiConnectors",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Prevents connector integrations from being available by default.",
	},
	{
		ID:          "permissions.deny",
		Group:       "Permissions",
		Title:       "Deny WebFetch",
		Description: "Adds WebFetch to the Claude Code permissions deny list.",
		Key:         "permissions.deny",
		Desired:     []string{"WebFetch"},
		DefaultSafe: false,
		Impact:      "Prevents Claude Code from using WebFetch unless this deny rule is removed.",
		MergeArray:  true,
	},
}
