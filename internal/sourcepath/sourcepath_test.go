package sourcepath

import (
	"reflect"
	"runtime"
	"testing"
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
