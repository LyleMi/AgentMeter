package viewmodel

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

// FormatNumber formats whole-number counters with comma group separators.
func FormatNumber(value int64) string {
	raw := strconv.FormatInt(value, 10)
	negative := strings.HasPrefix(raw, "-")
	if negative {
		raw = strings.TrimPrefix(raw, "-")
	}
	if len(raw) <= 3 {
		if negative {
			return "-" + raw
		}
		return raw
	}

	var builder strings.Builder
	if negative {
		builder.WriteByte('-')
	}
	prefix := len(raw) % 3
	if prefix == 0 {
		prefix = 3
	}
	builder.WriteString(raw[:prefix])
	for index := prefix; index < len(raw); index += 3 {
		builder.WriteByte(',')
		builder.WriteString(raw[index : index+3])
	}
	return builder.String()
}

// FormatRatio formats a non-negative ratio with at most one fractional digit.
func FormatRatio(value float64) string {
	if !isFinite(value) || value < 0 {
		value = 0
	}
	return trimFixed(roundTo(value, 1), 1, 0)
}

// FormatPercent formats a ratio as a whole percentage, clamping negative and
// non-finite values to zero.
func FormatPercent(value float64) string {
	if !isFinite(value) || value < 0 {
		value = 0
	}
	return FormatNumber(int64(math.Round(value*100))) + "%"
}

// FormatCost formats a USD cost. Nil means pricing coverage is incomplete.
func FormatCost(value *float64) string {
	if value == nil {
		return "unpriced"
	}
	return FormatCostValue(*value)
}

// FormatCostValue formats a concrete USD value with currency-style precision:
// at least two and at most four fractional digits.
func FormatCostValue(value float64) string {
	if !isFinite(value) {
		value = 0
	}
	negative := value < 0
	if negative {
		value = -value
	}
	formatted := trimFixed(roundTo(value, 4), 4, 2)
	if negative {
		return "-$" + formatted
	}
	return "$" + formatted
}

// FormatCostPerThousand formats cost density for token counts.
func FormatCostPerThousand(cost *float64, tokens int64) string {
	if cost == nil {
		return "unpriced"
	}
	if tokens <= 0 {
		return "$0"
	}
	value := *cost / (float64(tokens) / 1000)
	return FormatCostValue(value)
}

// FormatDuration formats a millisecond duration using the compact UI shape.
func FormatDuration(ms float64) string {
	if !isFinite(ms) || ms < 0 {
		ms = 0
	}
	total := int64(math.Round(ms / 1000))
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	if hours > 0 {
		return FormatNumber(hours) + "h " + FormatNumber(minutes) + "m"
	}
	if minutes > 0 {
		return FormatNumber(minutes) + "m " + FormatNumber(seconds) + "s"
	}
	return FormatNumber(seconds) + "s"
}

// FormatDateTime formats a timestamp for compact table cells.
func FormatDateTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format("Jan 02, 15:04")
}

// ShortPath keeps the last three path segments and preserves short paths.
func ShortPath(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	parts := splitPath(value)
	if len(parts) <= 3 {
		return value
	}
	return ".../" + strings.Join(parts[len(parts)-3:], "/")
}

// SessionLabel returns the stable user-facing session identity.
func SessionLabel(session model.Session) string {
	if label := strings.TrimSpace(session.SessionKey); label != "" {
		return label
	}
	if label := strings.TrimSpace(session.CodexSessionID); label != "" {
		return label
	}
	return "#" + strconv.FormatInt(session.ID, 10)
}

func splitPath(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			parts = append(parts, field)
		}
	}
	return parts
}

func trimFixed(value float64, maxDigits, minDigits int) string {
	raw := strconv.FormatFloat(value, 'f', maxDigits, 64)
	dot := strings.IndexByte(raw, '.')
	if dot < 0 {
		if minDigits <= 0 {
			return raw
		}
		return raw + "." + strings.Repeat("0", minDigits)
	}
	for len(raw)-dot-1 > minDigits && strings.HasSuffix(raw, "0") {
		raw = strings.TrimSuffix(raw, "0")
	}
	if strings.HasSuffix(raw, ".") {
		if minDigits <= 0 {
			raw = strings.TrimSuffix(raw, ".")
		} else {
			raw += strings.Repeat("0", minDigits)
		}
	}
	return addIntegerGrouping(raw)
}

func addIntegerGrouping(value string) string {
	sign := ""
	if strings.HasPrefix(value, "-") {
		sign = "-"
		value = strings.TrimPrefix(value, "-")
	}
	integer := value
	fraction := ""
	if dot := strings.IndexByte(value, '.'); dot >= 0 {
		integer = value[:dot]
		fraction = value[dot:]
	}
	grouped := FormatNumber(mustParseInt(integer))
	if sign != "" && !strings.HasPrefix(grouped, "-") {
		grouped = sign + grouped
	}
	return grouped + fraction
}

func mustParseInt(value string) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func roundTo(value float64, digits int) float64 {
	factor := math.Pow10(digits)
	return math.Round(value*factor) / factor
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
