package tui

import (
	"context"
	"fmt"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) handlePrivacyKey(k keyMsg) (command, bool, bool) {
	if s.privacyPending != nil {
		return s.handlePendingPrivacyKey(k)
	}
	if s.privacyApplying && isPrivacyApplyKey(k) {
		s.status = "privacy profile already applying"
		return nil, false, true
	}
	if cmd, quit, handled := s.handlePrivacyNavigationKey(k); handled {
		return cmd, quit, handled
	}
	if k.typ == keyRune {
		return s.handlePrivacyRuneKey(k.ch)
	}
	return nil, false, false
}

func (s *state) handlePendingPrivacyKey(k keyMsg) (command, bool, bool) {
	action := *s.privacyPending
	switch k.typ {
	case keyEnter:
		s.privacyPending = nil
		return s.applyPrivacyProfile(action), false, true
	case keyEsc:
		s.privacyPending = nil
		s.status = fmt.Sprintf("cancelled %s profile for %s", action.profile, action.targetName)
		return nil, false, true
	case keyRune:
		if k.ch == 'q' || k.ch == 'Q' {
			return nil, true, true
		}
	}
	s.status = fmt.Sprintf("pending %s profile for %s; Enter writes config, Esc cancels", action.profile, action.targetName)
	return nil, false, true
}

func isPrivacyApplyKey(k keyMsg) bool {
	return k.typ == keyEnter || (k.typ == keyRune && isPrivacyProfileKey(k.ch))
}

func (s *state) handlePrivacyNavigationKey(k keyMsg) (command, bool, bool) {
	switch k.typ {
	case keyEnter:
		s.queuePrivacyProfile("recommended")
	case keyUp:
		s.movePrivacyTarget(-1)
	case keyDown:
		s.movePrivacyTarget(1)
	case keyHome:
		s.movePrivacyTargetTo(0)
	case keyEnd:
		s.movePrivacyTargetTo(len(s.privacy) - 1)
	case keyPageUp:
		s.movePrivacyDetail(-s.pageStep())
	case keyPageDown:
		s.movePrivacyDetail(s.pageStep())
	default:
		return nil, false, false
	}
	return nil, false, true
}

func (s *state) handlePrivacyRuneKey(ch rune) (command, bool, bool) {
	switch ch {
	case '[', 'k', 'K':
		s.movePrivacyTarget(-1)
	case ']', 'j', 'J':
		s.movePrivacyTarget(1)
	case 'a':
		s.queuePrivacyProfile("recommended")
	case 'A':
		s.queuePrivacyProfile("strict")
	case 'u':
		s.queuePrivacyProfile("default")
	default:
		return nil, false, false
	}
	return nil, false, true
}

func isPrivacyProfileKey(ch rune) bool {
	switch ch {
	case 'a', 'A', 'u', 'U':
		return true
	default:
		return false
	}
}

func (s *state) queuePrivacyProfile(profile string) {
	status := s.selectedPrivacyStatus()
	if status == nil {
		s.status = "no privacy target loaded"
		return
	}
	target := strings.TrimSpace(status.Target)
	if target == "" {
		s.status = "selected privacy target has no target id"
		return
	}
	action := privacyProfileAction{
		target:     target,
		targetName: privacyDisplayName(*status),
		profile:    profile,
	}
	s.privacyPending = &action
	s.err = nil
	s.status = fmt.Sprintf("confirm %s profile for %s with Enter; Esc cancels", profile, action.targetName)
}

func (s *state) applyPrivacyProfile(action privacyProfileAction) command {
	applier, ok := s.service.(privacyProfileApplier)
	if !ok {
		s.err = nil
		s.status = "privacy profile operations are not available in this build"
		return nil
	}
	s.err = nil
	s.privacyApplying = true
	s.status = fmt.Sprintf("applying %s profile to %s...", action.profile, action.targetName)
	return func(ctx context.Context, ch chan<- message) {
		result, err := applier.ApplyPrivacyProfile(action.target, action.profile)
		sendMessage(ctx, ch, privacyProfileMsg{
			target:     action.target,
			targetName: action.targetName,
			profile:    action.profile,
			result:     result,
			err:        err,
		})
	}
}

func (s *state) movePrivacyTarget(delta int) {
	if len(s.privacy) == 0 {
		s.privacyTarget = 0
		s.status = "no privacy target loaded"
		return
	}
	next := s.privacyTarget + delta
	if next < 0 {
		next = len(s.privacy) - 1
	}
	if next >= len(s.privacy) {
		next = 0
	}
	s.setPrivacyTarget(next)
}

func (s *state) movePrivacyTargetTo(index int) {
	if len(s.privacy) == 0 {
		s.privacyTarget = 0
		s.status = "no privacy target loaded"
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(s.privacy) {
		index = len(s.privacy) - 1
	}
	s.setPrivacyTarget(index)
}

func (s *state) setPrivacyTarget(index int) {
	s.privacyTarget = index
	s.scroll = 0
	s.status = "selected privacy target: " + privacyDisplayName(s.privacy[s.privacyTarget])
}

func (s *state) movePrivacyDetail(delta int) {
	if len(s.privacy) == 0 {
		s.scroll = 0
		s.status = "no privacy target loaded"
		return
	}
	maxScroll := s.maxScroll()
	s.scroll += delta
	if s.scroll < 0 {
		s.scroll = 0
	}
	if s.scroll > maxScroll {
		s.scroll = maxScroll
	}
	if maxScroll == 0 {
		s.status = "selected privacy target fits on screen"
		return
	}
	s.status = fmt.Sprintf("privacy detail scroll %d/%d for %s", s.scroll+1, maxScroll+1, privacyDisplayName(s.privacy[s.privacyTarget]))
}

func (s *state) selectedPrivacyStatus() *agentmodel.PrivacyConfigStatus {
	if len(s.privacy) == 0 {
		return nil
	}
	s.clampPrivacyTarget()
	return &s.privacy[s.privacyTarget]
}

func (s *state) clampPrivacyTarget() {
	if len(s.privacy) == 0 {
		s.privacyTarget = 0
		return
	}
	if s.privacyTarget < 0 {
		s.privacyTarget = 0
	}
	if s.privacyTarget >= len(s.privacy) {
		s.privacyTarget = len(s.privacy) - 1
	}
}

func (s *state) mergePrivacyStatus(status agentmodel.PrivacyConfigStatus, fallbackTarget string) {
	target := strings.TrimSpace(status.Target)
	if target == "" {
		target = strings.TrimSpace(fallbackTarget)
	}
	if target == "" {
		return
	}
	for i := range s.privacy {
		if strings.EqualFold(s.privacy[i].Target, target) {
			if status.Target == "" {
				status.Target = s.privacy[i].Target
			}
			s.privacy[i] = status
			s.privacyTarget = i
			return
		}
	}
	if status.Target == "" {
		status.Target = target
	}
	s.privacy = append(s.privacy, status)
	s.privacyTarget = len(s.privacy) - 1
}

func privacyApplyStatus(profile, fallbackTargetName string, result agentmodel.PrivacyConfigApplyResult) string {
	targetName := privacyDisplayName(result.Status)
	if targetName == "unknown" && strings.TrimSpace(fallbackTargetName) != "" {
		targetName = fallbackTargetName
	}
	parts := []string{fmt.Sprintf("applied %s profile to %s", profile, targetName)}
	if len(result.Changed) == 1 {
		parts = append(parts, "1 change")
	} else {
		parts = append(parts, fmt.Sprintf("%d changes", len(result.Changed)))
	}
	if result.BackupPath != "" {
		parts = append(parts, "backup: "+result.BackupPath)
	}
	if len(result.Warnings) > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s): %s", len(result.Warnings), result.Warnings[0]))
	}
	return strings.Join(parts, "; ")
}

func privacyDisplayName(status agentmodel.PrivacyConfigStatus) string {
	name := strings.TrimSpace(empty(status.Name, status.Target))
	if name == "" {
		return "unknown"
	}
	return name
}
