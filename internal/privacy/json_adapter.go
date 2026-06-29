package privacy

import (
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type jsonAdapterSpec struct {
	target      string
	name        string
	agentName   string
	definitions []jsonSettingDefinition
	path        jsonSettingsPathSpec
}

type jsonSettingsPathSpec struct {
	overrideEnv  string
	configDirEnv string
	homeDirName  string
}

func (s jsonAdapterSpec) adapter(settingsPath func() (string, error), now func() time.Time) jsonPrivacyAdapter {
	return jsonPrivacyAdapter{
		target:       s.target,
		name:         s.name,
		agentName:    s.agentName,
		definitions:  s.definitions,
		settingsPath: settingsPath,
		now:          now,
	}
}

func (s jsonAdapterSpec) settingsPath(configPath string) (string, error) {
	return jsonSettingsPath(configPath, s.path.overrideEnv, s.path.configDirEnv, s.path.homeDirName)
}

func (s jsonAdapterSpec) settingsPathFunc(configPath string) func() (string, error) {
	return func() (string, error) {
		return s.settingsPath(configPath)
	}
}

func (s jsonAdapterSpec) buildStatus(path string, exists bool, content []byte, warnings []string) model.PrivacyConfigStatus {
	return s.adapter(nil, nil).buildStatus(path, exists, content, warnings)
}
