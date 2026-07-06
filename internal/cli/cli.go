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

type privacyCommandContext struct {
	args     []string
	stdout   io.Writer
	stderr   io.Writer
	registry privacyRegistry
}

func (c privacyCommandContext) withArgs(args []string) privacyCommandContext {
	c.args = args
	return c
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
	ctx := privacyCommandContext{
		args:     args,
		stdout:   stdout,
		stderr:   stderr,
		registry: registry,
	}
	if len(ctx.args) == 0 {
		PrintUsage(stdout)
		return ExitOK
	}

	switch normalizeCommand(ctx.args[0]) {
	case "help", "-h", "--help":
		PrintUsage(stdout)
		return ExitOK
	case "privacy":
		return ctx.withArgs(ctx.args[1:]).runPrivacy()
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", ctx.args[0])
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

func (c privacyCommandContext) runPrivacy() int {
	if len(c.args) == 0 {
		printPrivacyUsage(c.stdout)
		return ExitOK
	}

	switch normalizeCommand(c.args[0]) {
	case "help", "-h", "--help":
		printPrivacyUsage(c.stdout)
		return ExitOK
	case "targets", "target", "list", "ls":
		return c.withArgs(c.args[1:]).runPrivacyTargets()
	case "status", "summary":
		return c.withArgs(c.args[1:]).runPrivacyStatus()
	case "settings", "setting":
		return c.withArgs(c.args[1:]).runPrivacySettings()
	case "apply":
		return c.withArgs(c.args[1:]).runPrivacyApply()
	case "harden":
		if len(c.args) != 2 {
			fmt.Fprintln(c.stderr, "usage: agentmeter privacy harden <target|all>")
			return ExitUsage
		}
		return c.withArgs([]string{c.args[1], "strict"}).runPrivacyApply()
	case "set":
		return c.withArgs(c.args[1:]).runPrivacySet()
	case "unset":
		return c.withArgs(c.args[1:]).runPrivacyUnset()
	default:
		fmt.Fprintf(c.stderr, "unknown privacy command %q\n\n", c.args[0])
		printPrivacyUsage(c.stderr)
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

func (c privacyCommandContext) runPrivacyTargets() int {
	if len(c.args) != 0 {
		fmt.Fprintln(c.stderr, "privacy targets does not accept arguments")
		return ExitUsage
	}
	for _, target := range c.registry.Targets() {
		fmt.Fprintln(c.stdout, target)
	}
	return ExitOK
}

func (c privacyCommandContext) runPrivacyStatus() int {
	if len(c.args) > 1 {
		fmt.Fprintln(c.stderr, "usage: agentmeter privacy status [target|all]")
		return ExitUsage
	}

	target := "all"
	if len(c.args) == 1 {
		target = normalizeCommand(c.args[0])
	}
	if isAllTarget(target) {
		return c.runPrivacyStatusAll()
	}

	status, err := c.registry.Status(target)
	if err != nil {
		return printError(c.stderr, err)
	}
	printStatus(c.stdout, status)
	return ExitOK
}

func (c privacyCommandContext) runPrivacyStatusAll() int {
	code := ExitOK
	first := true
	for _, target := range c.registry.Targets() {
		status, err := c.registry.Status(target)
		if err != nil {
			code = ExitError
			fmt.Fprintf(c.stderr, "%s: %v\n", target, err)
			continue
		}
		if !first {
			fmt.Fprintln(c.stdout)
		}
		first = false
		printStatus(c.stdout, status)
	}
	return code
}

func (c privacyCommandContext) runPrivacySettings() int {
	if len(c.args) != 1 || isAllTarget(c.args[0]) {
		fmt.Fprintln(c.stderr, "usage: agentmeter privacy settings <target>")
		return ExitUsage
	}

	status, err := c.registry.Status(normalizeCommand(c.args[0]))
	if err != nil {
		return printError(c.stderr, err)
	}
	printStatus(c.stdout, status)
	if len(status.Settings) == 0 {
		fmt.Fprintln(c.stdout, "Settings: none")
		return ExitOK
	}
	fmt.Fprintln(c.stdout, "Settings:")
	for _, setting := range status.Settings {
		current := "unset"
		if setting.Configured {
			current = formatValue(setting.CurrentValue)
		}
		strict := setting.StrictValue
		if strict == nil {
			strict = setting.DesiredValue
		}
		fmt.Fprintf(c.stdout, "  - %s [%s] current=%s strict=%s\n", setting.ID, setting.Status, current, formatValue(strict))
	}
	return ExitOK
}

func (c privacyCommandContext) runPrivacyApply() int {
	if len(c.args) < 1 {
		fmt.Fprintln(c.stderr, "usage: agentmeter privacy apply <target|all> [recommended|strict|default|setting-id...]")
		return ExitUsage
	}

	target := normalizeCommand(c.args[0])
	rest := c.args[1:]
	profile, settingIDs := applyOperation(rest)
	if isAllTarget(target) && len(settingIDs) > 0 {
		fmt.Fprintln(c.stderr, "agentmeter privacy apply all only supports profiles")
		return ExitUsage
	}
	if isAllTarget(target) {
		return c.runPrivacyApplyAll(profile)
	}
	return runPrivacyApplyOne(privacyApplyOneRequest{
		target:     target,
		profile:    profile,
		settingIDs: settingIDs,
		stdout:     c.stdout,
		stderr:     c.stderr,
		registry:   c.registry,
	})
}

func (c privacyCommandContext) runPrivacyApplyAll(profile string) int {
	code := ExitOK
	for index, target := range c.registry.Targets() {
		if index > 0 {
			fmt.Fprintln(c.stdout)
		}
		if err := applyProfile(target, profile, c.stdout, c.registry); err != nil {
			code = ExitError
			fmt.Fprintf(c.stderr, "%s: %v\n", target, err)
		}
	}
	return code
}

type privacyApplyOneRequest struct {
	target     string
	profile    string
	settingIDs []string
	stdout     io.Writer
	stderr     io.Writer
	registry   privacyRegistry
}

func runPrivacyApplyOne(req privacyApplyOneRequest) int {
	if len(req.settingIDs) > 0 {
		result, err := req.registry.Apply(req.target, req.settingIDs)
		if err != nil {
			return printError(req.stderr, err)
		}
		printApplyResult(req.stdout, fmt.Sprintf("Applied %d setting(s)", len(req.settingIDs)), result)
		return ExitOK
	}
	if err := applyProfile(req.target, req.profile, req.stdout, req.registry); err != nil {
		return printError(req.stderr, err)
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

func (c privacyCommandContext) runPrivacySet() int {
	if len(c.args) < 3 || isAllTarget(c.args[0]) {
		fmt.Fprintln(c.stderr, "usage: agentmeter privacy set <target> <setting-id> <json-or-string-value>")
		return ExitUsage
	}
	value, err := parseConfigValue(strings.Join(c.args[2:], " "))
	if err != nil {
		return printError(c.stderr, err)
	}
	result, err := c.registry.ApplyChanges(normalizeCommand(c.args[0]), []model.PrivacyConfigEdit{{
		ID:    c.args[1],
		Op:    "set",
		Value: value,
	}})
	if err != nil {
		return printError(c.stderr, err)
	}
	printApplyResult(c.stdout, "Set privacy setting", result)
	return ExitOK
}

func (c privacyCommandContext) runPrivacyUnset() int {
	if len(c.args) != 2 || isAllTarget(c.args[0]) {
		fmt.Fprintln(c.stderr, "usage: agentmeter privacy unset <target> <setting-id>")
		return ExitUsage
	}
	result, err := c.registry.ApplyChanges(normalizeCommand(c.args[0]), []model.PrivacyConfigEdit{{
		ID: c.args[1],
		Op: "unset",
	}})
	if err != nil {
		return printError(c.stderr, err)
	}
	printApplyResult(c.stdout, "Unset privacy setting", result)
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
