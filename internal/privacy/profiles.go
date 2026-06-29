package privacy

import (
	"fmt"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

const (
	privacyProfileDefault     = "default"
	privacyProfileRecommended = "recommended"
	privacyProfileStrict      = "strict"

	privacyProfileOpSet   = "set"
	privacyProfileOpUnset = "unset"
)

type UnsupportedProfileError struct {
	Profile string
}

func (e UnsupportedProfileError) Error() string {
	return fmt.Sprintf("unsupported privacy profile: %s", e.Profile)
}

func normalizePrivacyProfile(profile string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(profile))
	switch normalized {
	case privacyProfileDefault, privacyProfileRecommended, privacyProfileStrict:
		return normalized, nil
	default:
		return "", UnsupportedProfileError{Profile: profile}
	}
}

func privacyConfigProfiles() []model.PrivacyConfigProfile {
	return []model.PrivacyConfigProfile{
		{
			ID:          privacyProfileDefault,
			Title:       "Default",
			Description: "Use vendor defaults by unsetting AgentMeter-managed privacy settings.",
		},
		{
			ID:          privacyProfileRecommended,
			Title:       "Recommended",
			Description: "Disable telemetry and reporting while leaving local retention, memory, and network controls at vendor defaults.",
		},
		{
			ID:          privacyProfileStrict,
			Title:       "Strict",
			Description: "Apply every AgentMeter-managed privacy hardening setting.",
		},
	}
}

func privacyProfileOperation(profile string, recommended bool) string {
	switch profile {
	case privacyProfileStrict:
		return privacyProfileOpSet
	case privacyProfileRecommended:
		if recommended {
			return privacyProfileOpSet
		}
		return privacyProfileOpUnset
	default:
		return privacyProfileOpUnset
	}
}

func privacyProfileValues(recommended bool, recommendedValue, strictValue any) []model.PrivacyConfigProfileValue {
	recommendedProfile := model.PrivacyConfigProfileValue{
		Profile: privacyProfileRecommended,
		Op:      privacyProfileOpUnset,
	}
	if recommended {
		recommendedProfile.Op = privacyProfileOpSet
		recommendedProfile.Value = cloneJSONValue(recommendedValue)
	}

	return []model.PrivacyConfigProfileValue{
		{
			Profile: privacyProfileDefault,
			Op:      privacyProfileOpUnset,
		},
		recommendedProfile,
		{
			Profile: privacyProfileStrict,
			Op:      privacyProfileOpSet,
			Value:   cloneJSONValue(strictValue),
		},
	}
}
