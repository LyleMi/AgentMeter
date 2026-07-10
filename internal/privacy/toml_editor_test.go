package privacy

import "testing"

func TestSplitValueCommentIgnoresQuotedHashes(t *testing.T) {
	tests := []struct {
		raw         string
		wantValue   string
		wantComment string
	}{
		{raw: `"value # retained" # comment`, wantValue: `"value # retained"`, wantComment: " # comment"},
		{raw: `'value # retained' # comment`, wantValue: `'value # retained'`, wantComment: " # comment"},
		{raw: `"value \"# retained" # comment`, wantValue: `"value \"# retained"`, wantComment: " # comment"},
		{raw: `true`, wantValue: "true", wantComment: ""},
	}

	for _, tt := range tests {
		value, comment := splitValueComment(tt.raw)
		if value != tt.wantValue || comment != tt.wantComment {
			t.Errorf("splitValueComment(%q) = %q, %q; want %q, %q", tt.raw, value, comment, tt.wantValue, tt.wantComment)
		}
	}
}

func TestIndexUnquotedIgnoresQuotedEquals(t *testing.T) {
	line := `key = "a=b"`
	if got := indexUnquoted(line, '='); got != 4 {
		t.Fatalf("indexUnquoted() = %d, want 4", got)
	}
}
