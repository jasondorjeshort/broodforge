// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// CodeWIP is worktree-wip's code: the tree's uncommitted changes could not be committed.
const CodeWIP = "worktree-wip"

// WIP is the one commit worktree-wip makes of a tree's uncommitted changes at a session's end:
// author and committer the driver, as every session commit is (03-architecture-git.md,
// Attribution), and the whole message, its trailer block included.
type WIP struct {
	Name, Email string
	Message     string
}

// WorktreeDirty reports whether the session's tree holds anything `git status --porcelain`
// lists: a change to a tracked file, staged or not, or an untracked file that is not ignored.
// Every other worktree inside the tree (a subagent's, `git worktree list`) is left out, and the
// status takes no optional lock, since a turn may be running in the tree.
func (s *SameAccount) WorktreeDirty(w Worktree) (bool, error) {
	if err := checkWorktree(w); err != nil {
		return false, err
	}
	scope, err := wipScope(w.Tree())
	if err != nil {
		return false, err
	}
	out, err := gitIn(nil, "", append([]string{"--no-optional-locks", "-C", w.Tree(), "status", "--porcelain", "-z", "--"}, scope...)...)
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// WorktreeWIP commits everything uncommitted in the session's tree as one commit on the session
// branch (`git add --all`, then `git commit --no-verify`) and returns its hash; a clean tree
// commits nothing and returns "". The worktrees inside the tree are left out of both, so a
// subagent's never becomes a gitlink. A HEAD on any other branch is refused, since the bundle
// carries the session branch alone. The commit skips the tree's hooks (the message already
// carries the trailers the commit-msg hook would add) and runs with core.hooksPath set to
// HooksDir, so no hook the tree configured runs as the bot.
func (s *SameAccount) WorktreeWIP(w Worktree, c WIP) (string, error) {
	dirty, err := s.WorktreeDirty(w)
	if err != nil || !dirty {
		return "", err
	}
	if c.Name == "" || c.Email == "" || strings.TrimSpace(c.Message) == "" {
		return "", &Error{CodeWIP, "a WIP commit needs the driver's name and email and a message"}
	}
	tree := w.Tree()
	head, err := runGit(CodeWIP, "-C", tree, "symbolic-ref", "--quiet", "HEAD")
	if err != nil || strings.TrimSpace(head) != "refs/heads/"+w.Branch {
		return "", &Error{CodeWIP, "the tree's HEAD is not on " + w.Branch + ": its uncommitted changes were not committed"}
	}
	scope, err := wipScope(tree)
	if err != nil {
		return "", err
	}
	safe := []string{"-C", tree, "-c", "core.hooksPath=" + s.HooksDir(), "-c", "commit.gpgsign=false"}
	if _, err := gitIn(nil, "", append(append(safe, "add", "--all", "--"), scope...)...); err != nil {
		return "", err
	}
	identity := []Var{{"GIT_AUTHOR_NAME", c.Name}, {"GIT_AUTHOR_EMAIL", c.Email},
		{"GIT_COMMITTER_NAME", c.Name}, {"GIT_COMMITTER_EMAIL", c.Email}}
	if _, err := gitIn(identity, c.Message, append(safe, "commit", "--quiet", "--no-verify", "--cleanup=verbatim", "--file=-")...); err != nil {
		return "", err
	}
	out, err := runGit(CodeWIP, "-C", tree, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// wipScope is the pathspec of the WIP status and add: the whole tree, then one
// `:(exclude,literal)<path>` for every other worktree of the clone that lies inside it (`git
// worktree list --porcelain`), each relative to the tree's top as git spells it.
func wipScope(tree string) ([]string, error) {
	top, err := runGit(CodeWIP, "-C", tree, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	top = strings.TrimSuffix(strings.TrimSpace(top), "/")
	list, err := runGit(CodeWIP, "-C", tree, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	scope := []string{"."}
	for _, line := range strings.Split(list, "\n") {
		path, ok := strings.CutPrefix(strings.TrimRight(line, "\r"), "worktree ")
		if !ok {
			continue
		}
		if rel, inside := cutDir(path, top); inside && rel != "" {
			scope = append(scope, ":(exclude,literal)"+rel)
		}
	}
	return scope, nil
}

// cutDir is path relative to dir when path is dir or under it; Windows paths compare without
// regard to case, as the file system does.
func cutDir(path, dir string) (string, bool) {
	p, d := strings.ToLower(path), strings.ToLower(dir)
	if p == d {
		return "", true
	}
	if strings.HasPrefix(p, d+"/") {
		return path[len(dir)+1:], true
	}
	return "", false
}

// gitIn is runGit with extra variables and stdin (message, "" for none); a failure names the
// subcommand past the global options.
func gitIn(vars []Var, message string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = Environment(os.Environ(), append([]Var{{"GIT_TERMINAL_PROMPT", "0"}}, vars...))
	if message != "" {
		cmd.Stdin = strings.NewReader(message)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		name := ""
		for i := 0; i < len(args); i++ {
			switch {
			case args[i] == "-C" || args[i] == "-c":
				i++
			case strings.HasPrefix(args[i], "--"):
			default:
				name = args[i]
			}
			if name != "" {
				break
			}
		}
		return "", &Error{CodeWIP, "git " + name + ": " + firstLine(msg)}
	}
	return stdout.String(), nil
}
