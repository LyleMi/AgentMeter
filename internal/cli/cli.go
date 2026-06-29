package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/privacy"
)

const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

type privacyRegistry interface {
	Targets() []string
	Status(target string) (model.PrivacyConfigStatus, error)
	Apply(target string, settingIDs []string) (model.PrivacyConfigApplyResult, error)
	ApplyChanges(target string, changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error)
	ApplyProfile(target, profile string) (model.PrivacyConfigApplyResult, error)
}

func IsCommand(command string) bool {
	switch normalizeCommand(command) {
	case "help", "privacy":
		return true
	default:
		return false
	}
}

func Run(args []string, stdout, stderr io.Writer) int {
	return runWithPrivacyRegistry(args, stdout, stderr, privacy.DefaultRegistry())
}

func runWithPrivacyRegistry(args []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(args) == 0 {
		PrintUsage(stdout)
		return ExitOK
	}

	switch normalizeCommand(args[0]) {
	case "help", "-h", "--help":
		PrintUsage(stdout)
		return ExitOK
	case "privacy":
		return runPrivacy(args[1:], stdout, stderr, registry)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		PrintUsage(stderr)
		return ExitUsage
	}
}

func PrintUsage(w io.Writer) {
	fmt.Fprint(w, `AgentMeter

Usage:
  agentmeter start [flags]
  agentmeter web [flags]
  agentmeter tui [flags]
  agentmeter privacy <command> [args]

Privacy commands:
  agentmeter privacy targets
  agentmeter privacy status [target|all]
  agentmeter privacy settings <target>
  agentmeter privacy apply <target|all> [recommended|strict|default]
  agentmeter privacy apply <target> <setting-id> [setting-id...]
  agentmeter privacy set <target> <setting-id> <json-or-string-value>
  agentmeter privacy unset <target> <setting-id>

Examples:
  agentmeter privacy status
  agentmeter privacy apply codex
  agentmeter privacy apply all recommended
  agentmeter privacy apply gemini strict
`)
}

func runPrivacy(args []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(args) == 0 {
		printPrivacyUsage(stdout)
		return ExitOK
	}

	switch normalizeCommand(args[0]) {
	case "help", "-h", "--help":
		printPrivacyUsage(stdout)
		return ExitOK
	case "targets", "target", "list", "ls":
		return runPrivacyTargets(args[1:], stdout, stderr, registry)
	case "status", "summary":
		return runPrivacyStatus(args[1:], stdout, stderr, registry)
	case "settings", "setting":
		return runPrivacySettings(args[1:], stdout, stderr, registry)
	case "apply":
		return runPrivacyApply(args[1:], stdout, stderr, registry)
	case "harden":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "usage: agentmeter privacy harden <target|all>")
			return ExitUsage
		}
		return runPrivacyApply([]string{args[1], "strict"}, stdout, stderr, registry)
	case "set":
		return runPrivacySet(args[1:], stdout, stderr, registry)
	case "unset":
		return runPrivacyUnset(args[1:], stdout, stderr, registry)
	default:
		fmt.Fprintf(stderr, "unknown privacy command %q\n\n", args[0])
		printPrivacyUsage(stderr)
		return ExitUsage
	}
}

func printPrivacyUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  agentmeter privacy targets
  agentmeter privacy status [target|all]
  agentmeter privacy settings <target>
  agentmeter privacy apply <target|all> [recommended|strict|default]
  agentmeter privacy apply <target> <setting-id> [setting-id...]
  agentmeter privacy set <target> <setting-id> <json-or-string-value>
  agentmeter privacy unset <target> <setting-id>

Targets:
  codex, gemini, claude, codebuddy, all

Profiles:
  recommended, strict, default

Examples:
  agentmeter privacy apply codex
  agentmeter privacy apply claude recommended
  agentmeter privacy apply all strict
  agentmeter privacy set codex web_search disabled
  agentmeter privacy unset gemini tools.exclude.web
`)
}

func runPrivacyTargets(args []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "privacy targets does not accept arguments")
		return ExitUsage
	}
	for _, target := range registry.Targets() {
		fmt.Fprintln(stdout, target)
	}
	return ExitOK
}

func runPrivacyStatus(args []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: agentmeter privacy status [target|all]")
		return ExitUsage
	}

	target := "all"
	if len(args) == 1 {
		target = normalizeCommand(args[0])
	}
	if isAllTarget(target) {
		return runPrivacyStatusAll(stdout, stderr, registry)
	}

	status, err := registry.Status(target)
	if err != nil {
		return printError(stderr, err)
	}
	printStatus(stdout, status)
	return ExitOK
}

func runPrivacyStatusAll(stdout, stderr io.Writer, registry privacyRegistry) int {
	code := ExitOK
	first := true
	for _, target := range registry.Targets() {
		status, err := registry.Status(target)
		if err != nil {
			code = ExitError
			fmt.Fprintf(stderr, "%s: %v\n", target, err)
			continue
		}
		if !first {
			fmt.Fprintln(stdout)
		}
		first = false
		printStatus(stdout, status)
	}
	return code
}

func runPrivacySettings(args []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(args) != 1 || isAllTarget(args[0]) {
		fmt.Fprintln(stderr, "usage: agentmeter privacy settings <target>")
		return ExitUsage
	}

	status, err := registry.Status(normalizeCommand(args[0]))
	if err != nil {
		return printError(stderr, err)
	}
	printStatus(stdout, status)
	if len(status.Settings) == 0 {
		fmt.Fprintln(stdout, "Settings: none")
		return ExitOK
	}
	fmt.Fprintln(stdout, "Settings:")
	for _, setting := range status.Settings {
		current := "unset"
		if setting.Configured {
			current = formatValue(setting.CurrentValue)
		}
		strict := setting.StrictValue
		if strict == nil {
			strict = setting.DesiredValue
		}
		fmt.Fprintf(stdout, "  - %s [%s] current=%s strict=%s\n", setting.ID, setting.Status, current, formatValue(strict))
	}
	return ExitOK
}

func runPrivacyApply(args []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: agentmeter privacy apply <target|all> [recommended|strict|default|setting-id...]")
		return ExitUsage
	}

	target := normalizeCommand(args[0])
	rest := args[1:]
	profile, settingIDs := applyOperation(rest)
	if isAllTarget(target) && len(settingIDs) > 0 {
		fmt.Fprintln(stderr, "agentmeter privacy apply all only supports profiles")
		return ExitUsage
	}
	if isAllTarget(target) {
		return runPrivacyApplyAll(profile, stdout, stderr, registry)
	}
	return runPrivacyApplyOne(target, profile, settingIDs, stdout, stderr, registry)
}

func runPrivacyApplyAll(profile string, stdout, stderr io.Writer, registry privacyRegistry) int {
	code := ExitOK
	for index, target := range registry.Targets() {
		if index > 0 {
			fmt.Fprintln(stdout)
		}
		if err := applyProfile(target, profile, stdout, registry); err != nil {
			code = ExitError
			fmt.Fprintf(stderr, "%s: %v\n", target, err)
		}
	}
	return code
}

func runPrivacyApplyOne(target, profile string, settingIDs []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(settingIDs) > 0 {
		result, err := registry.Apply(target, settingIDs)
		if err != nil {
			return printError(stderr, err)
		}
		printApplyResult(stdout, fmt.Sprintf("Applied %d setting(s)", len(settingIDs)), result)
		return ExitOK
	}
	if err := applyProfile(target, profile, stdout, registry); err != nil {
		return printError(stderr, err)
	}
	return ExitOK
}

func applyProfile(target, profile string, stdout io.Writer, registry privacyRegistry) error {
	result, err := registry.ApplyProfile(target, profile)
	if err != nil {
		return err
	}
	printApplyResult(stdout, fmt.Sprintf("Applied %s profile", profile), result)
	return nil
}

func runPrivacySet(args []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(args) < 3 || isAllTarget(args[0]) {
		fmt.Fprintln(stderr, "usage: agentmeter privacy set <target> <setting-id> <json-or-string-value>")
		return ExitUsage
	}
	value, err := parseConfigValue(strings.Join(args[2:], " "))
	if err != nil {
		return printError(stderr, err)
	}
	result, err := registry.ApplyChanges(normalizeCommand(args[0]), []model.PrivacyConfigEdit{{
		ID:    args[1],
		Op:    "set",
		Value: value,
	}})
	if err != nil {
		return printError(stderr, err)
	}
	printApplyResult(stdout, "Set privacy setting", result)
	return ExitOK
}

func runPrivacyUnset(args []string, stdout, stderr io.Writer, registry privacyRegistry) int {
	if len(args) != 2 || isAllTarget(args[0]) {
		fmt.Fprintln(stderr, "usage: agentmeter privacy unset <target> <setting-id>")
		return ExitUsage
	}
	result, err := registry.ApplyChanges(normalizeCommand(args[0]), []model.PrivacyConfigEdit{{
		ID: args[1],
		Op: "unset",
	}})
	if err != nil {
		return printError(stderr, err)
	}
	printApplyResult(stdout, "Unset privacy setting", result)
	return ExitOK
}

func applyOperation(args []string) (string, []string) {
	if len(args) == 0 {
		return "recommended", nil
	}
	if len(args) == 1 {
		profile := normalizeCommand(args[0])
		if isPrivacyProfile(profile) {
			return profile, nil
		}
	}
	return "", args
}

func printStatus(w io.Writer, status model.PrivacyConfigStatus) {
	fmt.Fprintf(w, "%s (%s)\n", status.Name, status.Target)
	state := "missing"
	if status.Exists {
		state = "exists"
	}
	fmt.Fprintf(w, "  file: %s [%s]\n", status.ConfigPath, state)
	fmt.Fprintf(
		w,
		"  score: %d%%; strict=%d/%d default-safe=%d needs-review=%d\n",
		status.Summary.Score,
		status.Summary.Hardened,
		status.Summary.Total,
		status.Summary.Implicit,
		status.Summary.Attention,
	)
	printWarnings(w, status.Warnings)
}

func printApplyResult(w io.Writer, action string, result model.PrivacyConfigApplyResult) {
	status := result.Status
	targetName := status.Name
	if targetName == "" {
		targetName = status.Target
	}
	fmt.Fprintf(w, "%s for %s (%s)\n", action, targetName, status.Target)
	if status.ConfigPath != "" {
		fmt.Fprintf(w, "  file: %s\n", status.ConfigPath)
	}
	if result.BackupPath != "" {
		fmt.Fprintf(w, "  backup: %s\n", result.BackupPath)
	}
	if len(result.Changed) == 0 {
		fmt.Fprintln(w, "  changed: 0")
		printWarnings(w, mergedWarnings(result.Warnings, status.Warnings))
		return
	}
	fmt.Fprintf(w, "  changed: %d\n", len(result.Changed))
	for _, change := range result.Changed {
		fmt.Fprintf(w, "  - %s: %s -> %s\n", change.ID, formatValue(change.Before), formatValue(change.After))
	}
	printWarnings(w, mergedWarnings(result.Warnings, status.Warnings))
}

func printWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		if strings.TrimSpace(warning) != "" {
			fmt.Fprintf(w, "  warning: %s\n", warning)
		}
	}
}

func parseConfigValue(raw string) (any, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}

	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	var parsed any
	if err := decoder.Decode(&parsed); err == nil {
		var extra any
		if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
			return parsed, nil
		}
	}
	return value, nil
}

func formatValue(value any) string {
	if value == nil {
		return "unset"
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

func printError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: %v\n", err)
	return ExitError
}

func mergedWarnings(left, right []string) []string {
	if len(left) == 0 {
		return right
	}
	if len(right) == 0 {
		return left
	}
	seen := make(map[string]struct{}, len(left)+len(right))
	result := make([]string, 0, len(left)+len(right))
	for _, value := range append(append([]string(nil), left...), right...) {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeCommand(command string) string {
	return strings.ToLower(strings.TrimSpace(command))
}

func isAllTarget(target string) bool {
	return normalizeCommand(target) == "all"
}

func isPrivacyProfile(profile string) bool {
	switch normalizeCommand(profile) {
	case "default", "recommended", "strict":
		return true
	default:
		return false
	}
}
