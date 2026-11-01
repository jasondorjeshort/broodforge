// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"os"
	"strings"
	"testing"
)

// TestHostCommand: the binary's own path, then --root, each quoted when it holds a space.
func TestHostCommand(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	q := exe
	if strings.Contains(exe, " ") {
		q = `"` + exe + `"`
	}
	for root, want := range map[string]string{
		"C:/ProgramData/Broodforge/main": q + " --root C:/ProgramData/Broodforge/main",
		"C:/Program Data/bf":             q + ` --root "C:/Program Data/bf"`,
		"":                               q,
	} {
		if got := HostCommand(root); got != want {
			t.Errorf("HostCommand(%q) = %q, want %q", root, got, want)
		}
	}
	if got := quoteSpace(`C:\Program Files\bf\broodforge.exe`); got != `"C:\Program Files\bf\broodforge.exe"` {
		t.Errorf("quoteSpace = %s", got)
	}
}

// TestGitCredentialPromptText: the URL when known; without one the prompt says none is known.
func TestGitCredentialPromptText(t *testing.T) {
	if got, want := gitCredentialPrompt("https://example.com/o/r.git"), "Token for pushes to https://example.com/o/r.git (not shown; Enter to add later with setup git-credential):"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := gitCredentialPrompt(""), "Token for pushes (no repository URL yet) (not shown; Enter to add later with setup git-credential):"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
