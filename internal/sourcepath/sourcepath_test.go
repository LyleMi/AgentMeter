package sourcepath

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestNormalizeListCleansSkipsEmptyAndDedupes(t *testing.T) {
	got := NormalizeList([]string{
		" ./alpha/../beta ",
		"",
		"   ",
		"beta",
		"beta/.",
	})

	want := []string{Normalize("beta")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeList() = %#v, want %#v", got, want)
	}
}

func TestNormalizeListUsesPlatformCaseKey(t *testing.T) {
	got := NormalizeList([]string{`C:\Work\Project`, `c:\work\project`})
	if runtime.GOOS == "windows" {
		if len(got) != 1 {
			t.Fatalf("NormalizeList() len = %d on Windows, want 1: %#v", len(got), got)
		}
		return
	}
	if len(got) != 2 {
		t.Fatalf("NormalizeList() len = %d on %s, want 2: %#v", len(got), runtime.GOOS, got)
	}
}

func TestEqualNormalizesAndUsesPlatformCaseKey(t *testing.T) {
	if !Equal("beta/.", "beta") {
		t.Fatal("Equal() should match clean-equivalent paths")
	}

	got := Equal(`C:\Work\Project`, `c:\work\project`)
	if runtime.GOOS == "windows" && !got {
		t.Fatal("Equal() should be case-insensitive on Windows")
	}
	if runtime.GOOS != "windows" && got {
		t.Fatalf("Equal() should be case-sensitive on %s", runtime.GOOS)
	}
}

func TestDedupeKeyNormalizesScopeAndRelative(t *testing.T) {
	got := DedupeKey(" ./scope/../scope ", " sessions/./one.jsonl ")
	want := Normalize("scope") + "\x00" + Normalize("sessions/one.jsonl")
	if got != want {
		t.Fatalf("DedupeKey() = %q, want %q", got, want)
	}
}

func TestSourceEntriesFromPathsNormalizesDedupesAndSetsEnabled(t *testing.T) {
	got := SourceEntriesFromPaths([]string{
		" ./alpha/../beta ",
		"",
		"beta/.",
		"gamma",
	}, false)
	want := []model.SourceEntry{
		{Path: Normalize("beta"), Enabled: false},
		{Path: Normalize("gamma"), Enabled: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SourceEntriesFromPaths() = %#v, want %#v", got, want)
	}
}

func TestNormalizeSourceEntriesCleansLabelsAndDedupesByPathKey(t *testing.T) {
	got := NormalizeSourceEntries([]model.SourceEntry{
		{Path: " ./alpha/../beta ", Enabled: false, Label: " Nightly "},
		{Path: "   "},
		{Path: "beta/.", Enabled: true, Label: "Duplicate"},
		{Path: "gamma", Enabled: true, Label: "\tStable\n"},
	})
	want := []model.SourceEntry{
		{Path: Normalize("beta"), Enabled: false, Label: "Nightly"},
		{Path: Normalize("gamma"), Enabled: true, Label: "Stable"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeSourceEntries() = %#v, want %#v", got, want)
	}
}

func TestEnabledSourceEntriesAndPathsFilterAfterNormalization(t *testing.T) {
	entries := []model.SourceEntry{
		{Path: " ./alpha/../beta ", Enabled: false, Label: " Disabled "},
		{Path: "beta/.", Enabled: true, Label: "Duplicate"},
		{Path: "gamma", Enabled: true, Label: " Enabled "},
	}

	gotEntries := EnabledSourceEntries(entries)
	wantEntries := []model.SourceEntry{
		{Path: Normalize("gamma"), Enabled: true, Label: "Enabled"},
	}
	if !reflect.DeepEqual(gotEntries, wantEntries) {
		t.Fatalf("EnabledSourceEntries() = %#v, want %#v", gotEntries, wantEntries)
	}

	gotPaths := EnabledSourceEntryPaths(entries)
	wantPaths := []string{Normalize("gamma")}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("EnabledSourceEntryPaths() = %#v, want %#v", gotPaths, wantPaths)
	}
}
