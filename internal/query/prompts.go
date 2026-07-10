package query

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
)

const (
	defaultPromptSuggestionLimit = 50
	maxPromptSuggestionLimit     = 200
	defaultPromptSuggestionMin   = 2
	maxPromptExamples            = 5
	maxPromptOccurrenceScan      = 5000
	minPromptTextRunes           = 4
	maxPromptTextRunes           = 1600
	promptSimilarityThreshold    = 0.72
)

var ErrInvalidPrompt = errors.New("invalid prompt")

var trivialPromptTexts = map[string]struct{}{
	"continue":      {},
	"commit":        {},
	"commit now":    {},
	"commit please": {},
	"commit一下":      {},
	"commit吧":       {},
	"date":          {},
	"done":          {},
	"good":          {},
	"ls":            {},
	"yes":           {},
	"no":            {},
	"ok":            {},
	"okay":          {},
	"please commit": {},
	"pwd":           {},
	"retry":         {},
	"status":        {},
	"thanks":        {},
	"thank you":     {},
	"好的":            {},
	"可以":            {},
	"继续":            {},
	"提交一下":          {},
	"提交吧":           {},
	"谢谢":            {},
}

type promptOccurrence struct {
	EventID            int64
	SourceLine         int
	Timestamp          time.Time
	Text               string
	Normalized         string
	SessionID          int64
	SessionKey         string
	CodexSessionID     string
	ProjectPath        string
	SourceID           int64
	SourceKey          string
	SourceLabel        string
	SourceRootPath     string
	SourceSessionsPath string
	AgentKind          string
	AgentName          string
	RawSourcePath      string
}

type promptVariantGroup struct {
	key        string
	normalized string
	text       string
	length     int
	grams      map[string]struct{}
	count      int
	sessions   map[int64]struct{}
	first      time.Time
	last       time.Time
	examples   []promptOccurrence
}

type promptCluster struct {
	variants []*promptVariantGroup
	examples []promptOccurrence
	minLen   int
	maxLen   int
}

func (s *Service) PromptSuggestions(ctx context.Context, filters model.PromptSuggestionFilters) ([]model.PromptSuggestion, error) {
	occurrences, err := s.promptOccurrences(ctx, filters)
	if err != nil {
		return nil, err
	}
	if len(occurrences) == 0 {
		return []model.PromptSuggestion{}, nil
	}

	savedKeys, err := s.savedPromptSuggestionKeys(ctx)
	if err != nil {
		return nil, err
	}
	ignoredKeys, err := s.ignoredPromptSuggestionKeys(ctx)
	if err != nil {
		return nil, err
	}

	minCount := promptSuggestionMinCount(filters.MinCount)
	limit := promptSuggestionLimit(filters.Limit)
	clusters := clusterPromptVariants(promptVariantGroups(occurrences))

	suggestions := make([]model.PromptSuggestion, 0, len(clusters))
	for _, cluster := range clusters {
		suggestion := cluster.suggestion()
		if suggestion.Count < minCount {
			continue
		}
		if promptSuggestionHasKey(suggestion, savedKeys) {
			continue
		}
		if promptSuggestionHasKey(suggestion, ignoredKeys) {
			continue
		}
		suggestions = append(suggestions, suggestion)
	}
	sortPromptSuggestions(suggestions)
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	return suggestions, nil
}

func promptSuggestionHasKey(suggestion model.PromptSuggestion, keys map[string]struct{}) bool {
	if _, ok := keys[suggestion.Key]; ok {
		return true
	}
	for _, variant := range suggestion.Variants {
		if _, ok := keys[variant.Key]; ok {
			return true
		}
	}
	return false
}

func (s *Service) SavedPrompts(ctx context.Context) ([]model.SavedPrompt, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT id, title, content, source_suggestion_key, copy_count, last_copied_at, created_at, updated_at
		FROM saved_prompts
		ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.SavedPrompt{}
	for rows.Next() {
		item, err := scanSavedPrompt(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) SavePrompt(ctx context.Context, input model.SavedPromptInput) (model.SavedPrompt, error) {
	title, content, sourceSuggestionKey, err := normalizeSavedPromptInput(input)
	if err != nil {
		return model.SavedPrompt{}, err
	}
	now := db.FormatTime(time.Now().UTC())
	res, err := s.conn.ExecContext(ctx, `INSERT INTO saved_prompts
		(title, content, source_suggestion_key, copy_count, last_copied_at, created_at, updated_at)
		VALUES (?, ?, ?, 0, '', ?, ?)`, title, content, sourceSuggestionKey, now, now)
	if err != nil {
		return model.SavedPrompt{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.SavedPrompt{}, err
	}
	return s.savedPrompt(ctx, id)
}

func (s *Service) UpdateSavedPrompt(ctx context.Context, id int64, input model.SavedPromptInput) (model.SavedPrompt, error) {
	if id <= 0 {
		return model.SavedPrompt{}, sql.ErrNoRows
	}
	title, content, sourceSuggestionKey, err := normalizeSavedPromptInput(input)
	if err != nil {
		return model.SavedPrompt{}, err
	}
	res, err := s.conn.ExecContext(ctx, `UPDATE saved_prompts
		SET title = ?, content = ?, source_suggestion_key = ?, updated_at = ?
		WHERE id = ?`, title, content, sourceSuggestionKey, db.FormatTime(time.Now().UTC()), id)
	if err != nil {
		return model.SavedPrompt{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return model.SavedPrompt{}, err
	}
	if affected == 0 {
		return model.SavedPrompt{}, sql.ErrNoRows
	}
	return s.savedPrompt(ctx, id)
}

func (s *Service) DeleteSavedPrompt(ctx context.Context, id int64) error {
	if id <= 0 {
		return sql.ErrNoRows
	}
	res, err := s.conn.ExecContext(ctx, `DELETE FROM saved_prompts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Service) RecordPromptCopy(ctx context.Context, id int64) (model.SavedPrompt, error) {
	if id <= 0 {
		return model.SavedPrompt{}, sql.ErrNoRows
	}
	now := db.FormatTime(time.Now().UTC())
	res, err := s.conn.ExecContext(ctx, `UPDATE saved_prompts
		SET copy_count = copy_count + 1, last_copied_at = ?, updated_at = ?
		WHERE id = ?`, now, now, id)
	if err != nil {
		return model.SavedPrompt{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return model.SavedPrompt{}, err
	}
	if affected == 0 {
		return model.SavedPrompt{}, sql.ErrNoRows
	}
	return s.savedPrompt(ctx, id)
}

func (s *Service) IgnorePromptSuggestion(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return promptValidationError("suggestionKey is required")
	}
	_, err := s.conn.ExecContext(ctx, `INSERT INTO ignored_prompt_suggestions (suggestion_key, ignored_at)
		VALUES (?, ?)
		ON CONFLICT(suggestion_key) DO UPDATE SET ignored_at = excluded.ignored_at`, key, db.FormatTime(time.Now().UTC()))
	return err
}

func (s *Service) UnignorePromptSuggestion(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return promptValidationError("suggestionKey is required")
	}
	_, err := s.conn.ExecContext(ctx, `DELETE FROM ignored_prompt_suggestions WHERE suggestion_key = ?`, key)
	return err
}

func (s *Service) promptOccurrences(ctx context.Context, filters model.PromptSuggestionFilters) ([]promptOccurrence, error) {
	where := []string{"e.kind = 'user'"}
	args := []any{}
	where, args = appendSourceFilterWithAlias(where, args, filters.Agent, "src")
	where, args = appendProjectFilter(where, args, filters.Project, "sess.project_path")

	query := fmt.Sprintf(`SELECT
			e.id, e.source_line, e.timestamp, e.raw_json,
			sess.id, COALESCE(NULLIF(sess.session_key, ''), sess.codex_session_id), sess.codex_session_id, sess.project_path,
			src.id, src.root_path, src.sessions_path, src.kind, src.name, sf.path
		FROM events e
		JOIN sessions sess ON sess.id = e.session_id
		JOIN sources src ON src.id = sess.source_id
		JOIN source_files sf ON sf.id = e.source_file_id
		WHERE %s
		ORDER BY e.timestamp DESC, e.id DESC
		LIMIT ?`, whereClause(where))
	args = append(args, maxPromptOccurrenceScan)
	rows, err := s.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	search := normalizePromptText(filters.Search)
	var result []promptOccurrence
	for rows.Next() {
		var item promptOccurrence
		var timestamp string
		var rawJSON string
		if err := rows.Scan(
			&item.EventID,
			&item.SourceLine,
			&timestamp,
			&rawJSON,
			&item.SessionID,
			&item.SessionKey,
			&item.CodexSessionID,
			&item.ProjectPath,
			&item.SourceID,
			&item.SourceRootPath,
			&item.SourceSessionsPath,
			&item.AgentKind,
			&item.AgentName,
			&item.RawSourcePath,
		); err != nil {
			return nil, err
		}
		item.Text = promptTextFromRawJSON(rawJSON)
		item.Normalized = normalizePromptText(item.Text)
		if !isPromptTextCandidate(item.Text, item.Normalized) {
			continue
		}
		if search != "" && !strings.Contains(item.Normalized, search) {
			continue
		}
		item.Timestamp = db.ParseTime(timestamp)
		if item.AgentName == "" {
			item.AgentName = item.AgentKind
		}
		item.SourceKey, item.SourceLabel = sourceIdentity(item.SourceID, item.AgentName, item.AgentKind)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) savedPromptSuggestionKeys(ctx context.Context) (map[string]struct{}, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT source_suggestion_key FROM saved_prompts WHERE source_suggestion_key != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPromptSuggestionKeySet(rows)
}

func (s *Service) ignoredPromptSuggestionKeys(ctx context.Context) (map[string]struct{}, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT suggestion_key FROM ignored_prompt_suggestions WHERE suggestion_key != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPromptSuggestionKeySet(rows)
}

func scanPromptSuggestionKeySet(rows *sql.Rows) (map[string]struct{}, error) {
	result := map[string]struct{}{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		key = strings.TrimSpace(key)
		if key != "" {
			result[key] = struct{}{}
		}
	}
	return result, rows.Err()
}

func isPromptTextCandidate(text, normalized string) bool {
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return false
	}
	runeCount := len([]rune(normalized))
	if runeCount < minPromptTextRunes || runeCount > maxPromptTextRunes {
		return false
	}
	if _, ok := trivialPromptTexts[normalized]; ok {
		return false
	}
	if looksLikeLowValueCommand(normalized) {
		return false
	}
	if looksLikeConversationArtifact(normalized) {
		return false
	}
	if looksLikeContextDump(normalized) {
		return false
	}
	if looksLikeAgentInstructions(normalized) {
		return false
	}
	if looksLikeXMLPrompt(text, normalized) {
		return false
	}
	if looksLikeToolPromptDump(normalized) {
		return false
	}
	if looksLikeStructuredPromptDump(text, normalized) {
		return false
	}
	return true
}

func looksLikeLowValueCommand(normalized string) bool {
	cleaned := strings.NewReplacer(
		",", " ",
		"，", " ",
		".", " ",
		"。", " ",
		"!", " ",
		"！", " ",
		"吧", " ",
		"一下", " ",
	).Replace(normalized)
	parts := strings.Fields(cleaned)
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}
	for _, part := range parts {
		switch part {
		case "commit", "good", "now", "ok", "please", "提交":
		default:
			return false
		}
	}
	return true
}

func looksLikeConversationArtifact(normalized string) bool {
	if strings.HasPrefix(normalized, "[") && strings.HasSuffix(normalized, "]") {
		return containsAnyPromptTerm(normalized,
			"request interrupted by user",
			"interrupted by user",
			"conversation interrupted",
		)
	}
	return containsAnyPromptTerm(normalized,
		"[request interrupted by user]",
		"request interrupted by user",
	)
}

func looksLikeContextDump(normalized string) bool {
	if strings.HasPrefix(normalized, "## context usage") {
		return true
	}
	return containsAnyPromptTerm(normalized,
		"estimated usage by category",
		"tokens | percentage",
		"### memory files",
		"### skills",
		"skill | source | tokens",
	)
}

func looksLikeAgentInstructions(normalized string) bool {
	return containsAnyPromptTerm(normalized,
		"# agents.md instructions for",
		"<instructions>",
		"</instructions>",
		"repository guidelines",
		"project structure & module organization",
		"build, test, and development commands",
		"coding style & naming conventions",
		"testing guidelines",
		"commit & pull request guidelines",
		"agent-specific instructions",
	)
}

func looksLikeXMLPrompt(text, normalized string) bool {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") && strings.Count(trimmed, "<") >= 2 {
		return true
	}
	return containsAnyPromptTerm(normalized,
		"<summary>",
		"</summary>",
		"<conversation_history_summary",
		"</conversation_history_summary>",
		"<tool_result",
		"</tool_result>",
		"<function_call",
		"</function_call>",
		"<function_call_output",
		"</function_call_output>",
	)
}

func looksLikeToolPromptDump(normalized string) bool {
	return containsAnyPromptTerm(normalized,
		`"tool_calls"`,
		`"tool_call_id"`,
		`"function_call"`,
		`"function_call_output"`,
		`"tool_result"`,
		"raw_start_event_json",
		"raw_end_event_json",
		"tool_use_id",
		"recipient_name",
		"shell_command",
	)
}

func looksLikeStructuredPromptDump(text, normalized string) bool {
	trimmed := strings.TrimSpace(text)
	if ((strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))) &&
		json.Valid([]byte(trimmed)) {
		return true
	}
	if strings.Contains(normalized, "```") && len([]rune(normalized)) > 240 {
		return true
	}
	if strings.HasPrefix(normalized, "diff --git ") || strings.Contains(normalized, " @@ ") {
		return true
	}
	return false
}

func containsAnyPromptTerm(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func promptVariantGroups(occurrences []promptOccurrence) []*promptVariantGroup {
	byText := map[string]*promptVariantGroup{}
	for _, occurrence := range occurrences {
		group := byText[occurrence.Normalized]
		if group == nil {
			group = &promptVariantGroup{
				key:        promptSuggestionKey(occurrence.Normalized),
				normalized: occurrence.Normalized,
				text:       occurrence.Text,
				length:     len([]rune(occurrence.Normalized)),
				grams:      promptNGrams(occurrence.Normalized, 3),
				sessions:   map[int64]struct{}{},
			}
			byText[occurrence.Normalized] = group
		}
		group.count++
		group.sessions[occurrence.SessionID] = struct{}{}
		if group.first.IsZero() || occurrence.Timestamp.Before(group.first) {
			group.first = occurrence.Timestamp
		}
		if group.last.IsZero() || occurrence.Timestamp.After(group.last) {
			group.last = occurrence.Timestamp
		}
		group.examples = append(group.examples, occurrence)
	}

	groups := make([]*promptVariantGroup, 0, len(byText))
	for _, group := range byText {
		groups = append(groups, group)
	}
	sortPromptVariantGroups(groups)
	return groups
}

func clusterPromptVariants(groups []*promptVariantGroup) []promptCluster {
	var clusters []promptCluster
	for _, group := range groups {
		bestIndex := -1
		bestScore := 0.0
		for index := range clusters {
			if !clusters[index].couldMatch(group) {
				continue
			}
			score := clusters[index].similarity(group)
			if score > bestScore {
				bestIndex = index
				bestScore = score
			}
		}
		if bestIndex >= 0 && bestScore >= promptSimilarityThreshold {
			clusters[bestIndex].add(group)
			continue
		}
		cluster := promptCluster{}
		cluster.add(group)
		clusters = append(clusters, cluster)
	}
	return clusters
}

func (c *promptCluster) add(group *promptVariantGroup) {
	c.variants = append(c.variants, group)
	c.examples = append(c.examples, group.examples...)
	if c.minLen == 0 || group.length < c.minLen {
		c.minLen = group.length
	}
	if group.length > c.maxLen {
		c.maxLen = group.length
	}
	sortPromptVariantGroups(c.variants)
}

func (c promptCluster) couldMatch(group *promptVariantGroup) bool {
	if group == nil || group.length <= 0 || c.minLen <= 0 || c.maxLen <= 0 {
		return false
	}
	if group.length < c.minLen {
		return float64(group.length)/float64(c.minLen) >= 0.65
	}
	if group.length > c.maxLen {
		return float64(c.maxLen)/float64(group.length) >= 0.65
	}
	return true
}

func (c promptCluster) similarity(group *promptVariantGroup) float64 {
	best := 0.0
	for _, variant := range c.variants {
		if score := promptSimilarityGroups(variant, group); score > best {
			best = score
		}
	}
	return best
}

func (c promptCluster) suggestion() model.PromptSuggestion {
	sortPromptVariantGroups(c.variants)
	canonical := c.variants[0]
	latest := latestPromptOccurrence(c.examples)
	suggestion := model.PromptSuggestion{
		Key:                canonical.key,
		Text:               canonical.text,
		VariantCount:       len(c.variants),
		FirstUsedAt:        canonical.first,
		LastUsedAt:         canonical.last,
		MatchKind:          c.matchKind(),
		Confidence:         c.confidence(),
		SourceID:           latest.SourceID,
		SourceKey:          latest.SourceKey,
		SourceLabel:        latest.SourceLabel,
		SourceRootPath:     latest.SourceRootPath,
		SourceSessionsPath: latest.SourceSessionsPath,
		AgentKind:          latest.AgentKind,
		AgentName:          latest.AgentName,
		Examples:           promptExamples(c.examples),
	}
	sessions := map[int64]struct{}{}
	for _, variant := range c.variants {
		suggestion.Count += variant.count
		if suggestion.FirstUsedAt.IsZero() || variant.first.Before(suggestion.FirstUsedAt) {
			suggestion.FirstUsedAt = variant.first
		}
		if suggestion.LastUsedAt.IsZero() || variant.last.After(suggestion.LastUsedAt) {
			suggestion.LastUsedAt = variant.last
		}
		for sessionID := range variant.sessions {
			sessions[sessionID] = struct{}{}
		}
		suggestion.Variants = append(suggestion.Variants, model.PromptVariant{
			Key:          variant.key,
			Text:         variant.text,
			Count:        variant.count,
			SessionCount: len(variant.sessions),
			FirstUsedAt:  variant.first,
			LastUsedAt:   variant.last,
		})
	}
	suggestion.SessionCount = len(sessions)
	suggestion.Examples = nonNilSlice(suggestion.Examples)
	suggestion.Variants = nonNilSlice(suggestion.Variants)
	return suggestion
}

func (c promptCluster) matchKind() string {
	if len(c.variants) <= 1 {
		return "exact"
	}
	return "near"
}

func (c promptCluster) confidence() float64 {
	if len(c.variants) <= 1 {
		return 1
	}
	canonical := c.variants[0]
	confidence := 1.0
	for _, variant := range c.variants[1:] {
		score := promptSimilarityGroups(canonical, variant)
		if score < confidence {
			confidence = score
		}
	}
	if confidence < 0 {
		return 0
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}

func promptExamples(occurrences []promptOccurrence) []model.PromptExample {
	sorted := append([]promptOccurrence(nil), occurrences...)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].Timestamp.Equal(sorted[j].Timestamp) {
			return sorted[i].Timestamp.After(sorted[j].Timestamp)
		}
		return sorted[i].EventID > sorted[j].EventID
	})

	result := make([]model.PromptExample, 0, maxPromptExamples)
	usedEvents := map[int64]struct{}{}
	seenSessions := map[int64]struct{}{}
	for _, occurrence := range sorted {
		if len(result) >= maxPromptExamples {
			break
		}
		if _, ok := seenSessions[occurrence.SessionID]; ok {
			continue
		}
		result = append(result, promptExample(occurrence))
		usedEvents[occurrence.EventID] = struct{}{}
		seenSessions[occurrence.SessionID] = struct{}{}
	}
	for _, occurrence := range sorted {
		if len(result) >= maxPromptExamples {
			break
		}
		if _, ok := usedEvents[occurrence.EventID]; ok {
			continue
		}
		result = append(result, promptExample(occurrence))
		usedEvents[occurrence.EventID] = struct{}{}
	}
	return result
}

func promptExample(occurrence promptOccurrence) model.PromptExample {
	return model.PromptExample{
		Text:               occurrence.Text,
		EventID:            occurrence.EventID,
		SourceLine:         occurrence.SourceLine,
		Timestamp:          occurrence.Timestamp,
		SessionID:          occurrence.SessionID,
		SessionKey:         occurrence.SessionKey,
		CodexSessionID:     occurrence.CodexSessionID,
		ProjectPath:        occurrence.ProjectPath,
		SourceID:           occurrence.SourceID,
		SourceKey:          occurrence.SourceKey,
		SourceLabel:        occurrence.SourceLabel,
		SourceRootPath:     occurrence.SourceRootPath,
		SourceSessionsPath: occurrence.SourceSessionsPath,
		AgentKind:          occurrence.AgentKind,
		AgentName:          occurrence.AgentName,
		RawSourcePath:      occurrence.RawSourcePath,
	}
}

func latestPromptOccurrence(occurrences []promptOccurrence) promptOccurrence {
	var latest promptOccurrence
	for _, occurrence := range occurrences {
		if latest.EventID == 0 ||
			occurrence.Timestamp.After(latest.Timestamp) ||
			(occurrence.Timestamp.Equal(latest.Timestamp) && occurrence.EventID > latest.EventID) {
			latest = occurrence
		}
	}
	return latest
}

func sortPromptVariantGroups(groups []*promptVariantGroup) {
	sort.Slice(groups, func(i, j int) bool {
		left := groups[i]
		right := groups[j]
		if left.count != right.count {
			return left.count > right.count
		}
		if !left.last.Equal(right.last) {
			return left.last.After(right.last)
		}
		return left.normalized < right.normalized
	})
}

func sortPromptSuggestions(suggestions []model.PromptSuggestion) {
	sort.Slice(suggestions, func(i, j int) bool {
		left := suggestions[i]
		right := suggestions[j]
		if left.Count != right.Count {
			return left.Count > right.Count
		}
		if left.SessionCount != right.SessionCount {
			return left.SessionCount > right.SessionCount
		}
		if !left.LastUsedAt.Equal(right.LastUsedAt) {
			return left.LastUsedAt.After(right.LastUsedAt)
		}
		return left.Text < right.Text
	})
}

func promptSimilarity(left, right string) float64 {
	leftGroup := &promptVariantGroup{
		normalized: left,
		length:     len([]rune(left)),
		grams:      promptNGrams(left, 3),
	}
	rightGroup := &promptVariantGroup{
		normalized: right,
		length:     len([]rune(right)),
		grams:      promptNGrams(right, 3),
	}
	return promptSimilarityGroups(leftGroup, rightGroup)
}

func promptSimilarityGroups(leftGroup, rightGroup *promptVariantGroup) float64 {
	if leftGroup == nil || rightGroup == nil {
		return 0
	}
	left := leftGroup.normalized
	right := rightGroup.normalized
	if left == right {
		return 1
	}
	if left == "" || right == "" {
		return 0
	}
	shorter, longer, lengthRatio := promptSimilarityLengthWindow(leftGroup, rightGroup)
	if lengthRatio < minPromptSimilarityLengthRatio {
		return 0
	}
	if promptSubstringSimilarity(shorter, longer, lengthRatio) {
		return 0.95
	}
	return promptGramSimilarity(
		promptGroupNGrams(leftGroup),
		promptGroupNGrams(rightGroup),
	)
}

const (
	minPromptSimilarityLengthRatio = 0.65
	minPromptSubstringLengthRatio  = 0.72
	minPromptSubstringRunes        = 16
)

func promptSimilarityLengthWindow(leftGroup, rightGroup *promptVariantGroup) (string, string, float64) {
	leftLen := promptGroupLength(leftGroup)
	rightLen := promptGroupLength(rightGroup)
	shorterLen, longerLen := leftLen, rightLen
	shorter, longer := leftGroup.normalized, rightGroup.normalized
	if shorterLen > longerLen {
		shorterLen, longerLen = longerLen, shorterLen
		shorter, longer = longer, shorter
	}
	return shorter, longer, float64(shorterLen) / float64(longerLen)
}

func promptGroupLength(group *promptVariantGroup) int {
	if group.length > 0 {
		return group.length
	}
	return len([]rune(group.normalized))
}

func promptSubstringSimilarity(shorter, longer string, lengthRatio float64) bool {
	return len([]rune(shorter)) >= minPromptSubstringRunes &&
		lengthRatio >= minPromptSubstringLengthRatio &&
		strings.Contains(longer, shorter)
}

func promptGroupNGrams(group *promptVariantGroup) map[string]struct{} {
	if len(group.grams) > 0 {
		return group.grams
	}
	return promptNGrams(group.normalized, 3)
}

func promptGramSimilarity(leftGrams, rightGrams map[string]struct{}) float64 {
	if len(leftGrams) == 0 || len(rightGrams) == 0 {
		return 0
	}
	intersection := promptGramIntersection(leftGrams, rightGrams)
	union := len(leftGrams) + len(rightGrams) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func promptGramIntersection(leftGrams, rightGrams map[string]struct{}) int {
	intersection := 0
	for gram := range leftGrams {
		if _, ok := rightGrams[gram]; ok {
			intersection++
		}
	}
	return intersection
}

func promptNGrams(value string, n int) map[string]struct{} {
	runes := []rune(value)
	if len(runes) == 0 {
		return nil
	}
	if len(runes) <= n {
		return map[string]struct{}{string(runes): {}}
	}
	result := make(map[string]struct{}, len(runes)-n+1)
	for index := 0; index+n <= len(runes); index++ {
		result[string(runes[index:index+n])] = struct{}{}
	}
	return result
}

func promptTextFromRawJSON(rawJSON string) string {
	var raw map[string]any
	decoder := json.NewDecoder(strings.NewReader(rawJSON))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return ""
	}
	return collapsePromptWhitespace(promptTextFromEnvelope(raw))
}

func promptTextFromEnvelope(raw map[string]any) string {
	if payload, ok := raw["payload"].(map[string]any); ok {
		if text := promptTextFromPayload(payload); text != "" {
			return text
		}
	}

	topType := promptLowerString(raw["type"])
	role := promptLowerString(raw["role"])
	if topType == "user_message" || role == "user" {
		return promptTextFromFields(raw, "content", "message", "text", "input")
	}
	if topType == "user" {
		return promptTextFromTopLevelUser(raw)
	}
	return ""
}

func promptTextFromTopLevelUser(raw map[string]any) string {
	if message, ok := raw["message"].(map[string]any); ok {
		messageRole := promptLowerString(message["role"])
		if messageRole == "" || messageRole == "user" {
			if text := promptTextFromFields(message, "content", "message", "text", "input"); text != "" {
				return text
			}
		}
	}
	return promptTextFromFields(raw, "content", "message", "text", "input")
}

func promptTextFromPayload(payload map[string]any) string {
	payloadType := promptLowerString(payload["type"])
	role := promptLowerString(payload["role"])
	if payloadType == "user_message" || role == "user" {
		return promptTextFromFields(payload, "content", "message", "text", "input")
	}
	return ""
}

func promptTextFromFields(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		text := promptTextFromValue(payload[key])
		if text != "" {
			return text
		}
	}
	return ""
}

func promptTextFromValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		return promptTextFromContentItems(typed)
	case map[string]any:
		if promptIsToolOnlyContent(typed) {
			return ""
		}
		role := promptLowerString(typed["role"])
		if role != "" && role != "user" {
			return ""
		}
		return promptTextFromFields(typed, "content", "message", "text", "input", "summary")
	default:
		return ""
	}
}

func promptTextFromContentItems(items []any) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				parts = append(parts, typed)
			}
		case map[string]any:
			if promptIsToolOnlyContent(typed) {
				continue
			}
			text := promptTextFromFields(typed, "text", "content", "message", "input", "summary")
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, " ")
}

func promptIsToolOnlyContent(payload map[string]any) bool {
	switch promptLowerString(payload["type"]) {
	case "tool_result", "tool_use", "function_call", "function_call_output", "custom_tool_call", "custom_tool_call_output":
		return true
	default:
		return false
	}
}

func promptLowerString(value any) string {
	text, _ := value.(string)
	return strings.ToLower(strings.TrimSpace(text))
}

func normalizePromptText(value string) string {
	return strings.ToLower(collapsePromptWhitespace(value))
}

func collapsePromptWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func promptSuggestionKey(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return "prompt:" + hex.EncodeToString(sum[:])[:16]
}

func promptSuggestionLimit(limit int) int {
	if limit <= 0 {
		return defaultPromptSuggestionLimit
	}
	if limit > maxPromptSuggestionLimit {
		return maxPromptSuggestionLimit
	}
	return limit
}

func promptSuggestionMinCount(minCount int) int {
	if minCount <= 0 {
		return defaultPromptSuggestionMin
	}
	return minCount
}

func (s *Service) savedPrompt(ctx context.Context, id int64) (model.SavedPrompt, error) {
	row := s.conn.QueryRowContext(ctx, `SELECT id, title, content, source_suggestion_key, copy_count, last_copied_at, created_at, updated_at
		FROM saved_prompts WHERE id = ?`, id)
	return scanSavedPrompt(row)
}

func scanSavedPrompt(row interface{ Scan(dest ...any) error }) (model.SavedPrompt, error) {
	var item model.SavedPrompt
	var lastCopiedAt string
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&item.ID,
		&item.Title,
		&item.Content,
		&item.SourceSuggestionKey,
		&item.CopyCount,
		&lastCopiedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return model.SavedPrompt{}, err
	}
	if parsed := db.ParseTime(lastCopiedAt); !parsed.IsZero() {
		item.LastCopiedAt = &parsed
	}
	item.CreatedAt = db.ParseTime(createdAt)
	item.UpdatedAt = db.ParseTime(updatedAt)
	return item, nil
}

func normalizeSavedPromptInput(input model.SavedPromptInput) (string, string, string, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return "", "", "", promptValidationError("content is required")
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = promptTitleFromContent(content)
	}
	return title, content, strings.TrimSpace(input.SourceSuggestionKey), nil
}

func promptTitleFromContent(content string) string {
	title := collapsePromptWhitespace(content)
	runes := []rune(title)
	if len(runes) <= 80 {
		return title
	}
	return string(runes[:77]) + "..."
}

func promptValidationError(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidPrompt, message)
}
