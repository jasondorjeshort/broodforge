// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AskpassUser is the user name the askpass helper answers: GitHub takes any name with a token
// as the password, and this is the name its documentation gives.
const AskpassUser = "x-access-token"

// AskpassAnswer is the hidden `broodforge git-askpass <prompt>` that share/askpass.sh and
// share/askpass.cmd run for git: a `Username` prompt gets AskpassUser, a `Password` prompt the
// credential in CredentialEnv, which only the bot's push sets. Any other prompt, or a password
// prompt with none set, is refused: git reads no answer and the push fails with
// handoff-failed-push.
func AskpassAnswer(prompt string, lookup func(string) (string, bool)) (string, error) {
	switch {
	case strings.HasPrefix(prompt, "Username"):
		return AskpassUser, nil
	case !strings.HasPrefix(prompt, "Password"):
		return "", &Error{Code: CodeNoCredential, Message: "the helper answers Username and Password prompts only"}
	}
	v, ok := lookup(CredentialEnv)
	if !ok || v == "" {
		return "", &Error{Code: CodeNoCredential, Message: CredentialEnv + " is not set: this push carries no git credential"}
	}
	return v, nil
}

// CommitMsg is the hidden `broodforge git-commit-msg <file>` that share/hooks/commit-msg runs
// in a session's clone (03-architecture.md, Attribution): it appends each line of
// <sess>/home/trailers the message lacks, through `git interpret-trailers --if-exists
// addIfDifferent`. It is a help, not the enforcement, so a missing trailers file changes
// nothing. The session's directory is found from the clone's common git directory, which is
// <sess>/tree/.git for the tree and for a subagent worktree inside it alike. It runs inside the
// session with the hook's environment, never in the bot.
func CommitMsg(ctx context.Context, msgFile string) error {
	out, err := hookGit(ctx, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	common := filepath.Clean(strings.TrimSpace(out))
	sess := filepath.Dir(filepath.Dir(common))
	raw, err := os.ReadFile(filepath.FromSlash(TrailersPath(sess)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return &Error{Code: CodeCommitMsg, Message: err.Error()}
	}
	args := []string{"interpret-trailers", "--in-place", "--if-exists", "addIfDifferent"}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	n := 0
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			args = append(args, "--trailer", line)
			n++
		}
	}
	if n == 0 {
		return nil
	}
	_, err = hookGit(ctx, append(args, msgFile)...)
	return err
}

// hookGit runs the session's own git with the hook's environment as git gave it.
func hookGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := tellingLine(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", &Error{Code: CodeCommitMsg, Message: "git " + args[0] + ": " + msg}
	}
	return stdout.String(), nil
}
