package tui

import "testing"

func TestPrivacyNavigationKeysSelectTargets(t *testing.T) {
	_, st := loadPrivacyPage(t, 100, 12)

	assertPrivacyTarget(t, st, 0, "initial selection")

	st.scroll = 3
	assertNoPrivacyCommand(t, st, keyMsg{typ: keyDown}, "down")
	assertPrivacyTarget(t, st, 1, "down")
	assertPrivacyScroll(t, st, 0, "down resets detail scroll")

	st.scroll = 2
	assertNoPrivacyCommand(t, st, keyMsg{typ: keyUp}, "up")
	assertPrivacyTarget(t, st, 0, "up")
	assertPrivacyScroll(t, st, 0, "up resets detail scroll")

	st.scroll = 2
	assertNoPrivacyCommand(t, st, keyMsg{typ: keyRune, ch: 'j'}, "j")
	assertPrivacyTarget(t, st, 1, "j")
	assertPrivacyScroll(t, st, 0, "j resets detail scroll")

	st.scroll = 2
	assertNoPrivacyCommand(t, st, keyMsg{typ: keyRune, ch: 'k'}, "k")
	assertPrivacyTarget(t, st, 0, "k")
	assertPrivacyScroll(t, st, 0, "k resets detail scroll")

	st.scroll = 2
	assertNoPrivacyCommand(t, st, keyMsg{typ: keyEnd}, "end")
	assertPrivacyTarget(t, st, len(st.privacy)-1, "end")
	assertPrivacyScroll(t, st, 0, "end resets detail scroll")

	st.scroll = 2
	assertNoPrivacyCommand(t, st, keyMsg{typ: keyHome}, "home")
	assertPrivacyTarget(t, st, 0, "home")
	assertPrivacyScroll(t, st, 0, "home resets detail scroll")
}

func TestPrivacyPageKeysScrollSelectedTargetDetails(t *testing.T) {
	_, st := loadPrivacyPage(t, 100, 10)
	st.privacyTarget = 1
	st.scroll = 0

	assertNoPrivacyCommand(t, st, keyMsg{typ: keyPageDown}, "page down")
	assertPrivacyTarget(t, st, 1, "page down keeps target")
	if st.scroll == 0 {
		t.Fatal("page down did not scroll selected target details")
	}

	scrolled := st.scroll
	assertNoPrivacyCommand(t, st, keyMsg{typ: keyPageUp}, "page up")
	assertPrivacyTarget(t, st, 1, "page up keeps target")
	if st.scroll >= scrolled {
		t.Fatalf("page up scroll = %d, want less than %d", st.scroll, scrolled)
	}
}

func TestPrivacyEnterQueuesRecommendedProfileAndRequiresConfirmation(t *testing.T) {
	svc, st := loadPrivacyPage(t, 100, 20)

	cmd, quit := st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("enter unexpectedly quit")
	}
	if cmd != nil {
		t.Fatal("first enter returned an apply command")
	}
	if len(svc.privacyApplyCall) != 0 {
		t.Fatalf("privacy apply calls = %v, want none after first enter", svc.privacyApplyCall)
	}
	if st.privacyPending == nil {
		t.Fatal("first enter did not queue a pending privacy action")
	}
	if got := *st.privacyPending; got.target != "codex" || got.targetName != "Codex" || got.profile != "recommended" {
		t.Fatalf("pending action = %+v, want Codex recommended", got)
	}
	view := st.view()
	assertContains(t, view, "Pending:")
	assertContains(t, view, "recommended profile to Codex")

	cmd, quit = st.update(keyMsg{typ: keyEsc})
	if quit {
		t.Fatal("esc unexpectedly quit")
	}
	if cmd != nil {
		t.Fatal("esc returned a command")
	}
	if st.privacyPending != nil {
		t.Fatalf("esc left pending action = %+v", *st.privacyPending)
	}
	if len(svc.privacyApplyCall) != 0 {
		t.Fatalf("privacy apply calls = %v, want none after esc", svc.privacyApplyCall)
	}

	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("enter unexpectedly quit")
	}
	if cmd != nil {
		t.Fatal("first enter after cancel returned an apply command")
	}
	if st.privacyPending == nil {
		t.Fatal("enter after cancel did not queue a pending privacy action")
	}

	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("confirm enter unexpectedly quit")
	}
	if cmd == nil {
		t.Fatal("confirm enter did not return an apply command")
	}
	if !st.privacyApplying {
		t.Fatal("confirm enter did not enter applying state")
	}
	if len(svc.privacyApplyCall) != 0 {
		t.Fatalf("privacy apply calls = %v, want command not run yet", svc.privacyApplyCall)
	}

	msg := runCommand(t, cmd)
	cmd, quit = st.update(msg)
	if quit {
		t.Fatal("apply result unexpectedly quit")
	}
	if cmd != nil {
		t.Fatal("apply result returned an unexpected command")
	}
	if len(svc.privacyApplyCall) != 1 {
		t.Fatalf("privacy apply calls = %v, want one", svc.privacyApplyCall)
	}
	if got := svc.privacyApplyCall[0]; got.target != "codex" || got.profile != "recommended" {
		t.Fatalf("privacy apply call = %+v, want codex recommended", got)
	}
}

func TestPrivacyRenderHighlightsSelectedTargetSummaryAndDetails(t *testing.T) {
	_, st := loadPrivacyPage(t, 120, 40)
	st.privacyTarget = 1
	st.scroll = 0

	view := st.view()
	assertContains(t, view, "Agent Privacy")
	assertContains(t, view, "Codex")
	assertContains(t, view, "Gemini CLI")
	assertContains(t, view, "Claude Code")
	assertContains(t, view, "CodeBuddy")
	assertContains(t, view, "Selected: Gemini CLI")
	assertContains(t, view, "Next:")
	assertContains(t, view, `C:\Users\agent\.gemini\settings.json`)
	assertContains(t, view, "Usage statistics")
	assertContains(t, view, "[attention] Web tools")
	assertContains(t, view, "Broken JSON")
	assertContains(t, view, "read-only")
}

func loadPrivacyPage(t *testing.T, width, height int) (*fakeService, *state) {
	t.Helper()
	svc := sampleService()
	st := newState(svc, width, height)
	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '5'})
	if quit {
		t.Fatal("privacy page switch unexpectedly quit")
	}
	if cmd == nil {
		t.Fatal("privacy page switch did not return a load command")
	}
	cmd, quit = st.update(runCommand(t, cmd))
	if quit {
		t.Fatal("privacy page load unexpectedly quit")
	}
	if cmd != nil {
		t.Fatal("privacy page load returned an unexpected command")
	}
	if st.page != pagePrivacy {
		t.Fatalf("page = %v, want privacy", st.page)
	}
	if len(st.privacy) == 0 {
		t.Fatal("privacy page did not load sample targets")
	}
	return svc, st
}

func assertNoPrivacyCommand(t *testing.T, st *state, key keyMsg, label string) {
	t.Helper()
	cmd, quit := st.update(key)
	if quit {
		t.Fatalf("%s unexpectedly quit", label)
	}
	if cmd != nil {
		t.Fatalf("%s returned an unexpected command", label)
	}
}

func assertPrivacyTarget(t *testing.T, st *state, want int, label string) {
	t.Helper()
	if st.privacyTarget != want {
		t.Fatalf("%s privacy target = %d, want %d", label, st.privacyTarget, want)
	}
}

func assertPrivacyScroll(t *testing.T, st *state, want int, label string) {
	t.Helper()
	if st.scroll != want {
		t.Fatalf("%s scroll = %d, want %d", label, st.scroll, want)
	}
}
