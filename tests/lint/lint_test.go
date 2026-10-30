// SPDX-License-Identifier: GPL-3.0-or-later

// Package lint enforces the mechanical rules of .claude/rules/go.md, so `go test ./...` does:
// the SPDX first line of every .go file; host branching only where
// docs/design/03-architecture.md puts host differences; no `any` or empty interface outside
// internal/engine's raw stream layer; and the import graph of 03's Components table, held below
// as data. Packages and imports come from `go list -json ./...` (the `go` on PATH) for GOOS
// linux and windows, so a host file of either is read on both hosts; the file rules parse every
// .go file either listing names. Test files' imports are not part of the graph: the rule is the
// components' own code.
package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const spdx = "// SPDX-License-Identifier: GPL-3.0-or-later"

// row03 prefixes the name of the 03-architecture.md row an entry mirrors; a failure names it.
const row03 = "docs/design/03-architecture.md (Components), row "

// entry is one package's allowed imports of this module's packages, module-relative.
type entry struct {
	row     string // where the entry's rule is written; the failure message names it
	imports []string
}

// components maps 03's component names to their packages, internal/<component>.
func components(names ...string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "internal/" + n
	}
	return out
}

// graph is the allowed import graph: every package of the module, keyed module-relative. A
// package missing here, an import of one missing here, or an import its entry does not allow
// fails the test with the entry's row to edit. Standard library and third-party imports are not
// checked here (go.md's library rule is docs/reference/go-dependencies.md's).
var graph = map[string]entry{
	// 03 row `discord`: talks to `SessionService` (package session), `config`, `policy`.
	"internal/discord": {row03 + "`discord`", components("session", "config", "policy")},
	// 03 row `control`: `SessionService` (package session), `store`, `backup`, `policy`.
	"internal/control": {row03 + "`control`", components("session", "store", "backup", "policy")},
	// 03 row `session`: `engine`, `git`, `policy`, `credentials`, `store`, `audit`.
	"internal/session": {row03 + "`session`", components("engine", "git", "policy", "credentials", "store", "audit")},
	// 03 row `engine`: `spawn`, `credentials`.
	"internal/engine": {row03 + "`engine`", components("spawn", "credentials")},
	// 03 row `hook`: `policy` (the decision function it shares with `session`); nothing in the
	// bot process.
	"internal/hook": {row03 + "`hook`", components("policy")},
	// 03 row `policy`: nothing (pure).
	"internal/policy": {row03 + "`policy`", nil},
	// 03 row `credentials`: `spawn`, `store`, `audit`.
	"internal/credentials": {row03 + "`credentials`", components("spawn", "store", "audit")},
	// 03 row `store`: `audit` writes in a store transaction; the import is `audit` to `store`,
	// one direction, so `store` imports no component.
	"internal/store": {row03 + "`store`", nil},
	// 03 row `audit`: `store`.
	"internal/audit": {row03 + "`audit`", components("store")},
	// 03 row `git`: `spawn`, `credentials`, `audit`.
	"internal/git": {row03 + "`git`", components("spawn", "credentials", "audit")},
	// 03 row `workflow`: `spawn`.
	"internal/workflow": {row03 + "`workflow`", components("spawn")},
	// 03 row `spawn`: the OS.
	"internal/spawn": {row03 + "`spawn`", nil},
	// 03 row `backup`: `store`.
	"internal/backup": {row03 + "`backup`", components("store")},
	// 03 row `config`: `credentials`, `policy`.
	"internal/config": {row03 + "`config`", components("credentials", "policy")},
	// 03 row `setup`: `config`, `credentials`, `store`, `spawn`, `git`, `discord`, `control`; `audit`.
	"internal/setup": {row03 + "`setup`", components("config", "credentials", "store", "spawn", "git", "discord", "control", "audit")},
	// 03 row `cli`: `control`, and the exceptions go.md defers to, every component a standalone
	// or `(file)` subcommand lives in.
	"internal/cli": {row03 + "`cli`", components("control",
		"engine", "config", "hook", // engine run|record, config check|show, hook ...
		"store", "audit", "backup", "credentials", // the (file) commands, credential init, the handlers
		"spawn",    // spawn preflight; the standalone engine commands' backend
		"session",  // serve builds the service
		"discord",  // serve's adapter, discord render, setup's Discord client
		"git",      // the worktree handlers, git-askpass, git-commit-msg, git-pre-push
		"policy",   // the same
		"workflow", // workflow render|probe, the handoff's copy, serve's copy and probe
		"setup",    // the setup command
	)},
	// 03 row `cli`: "the one entry point `broodforge <command>`"; 08 P0, main dispatches to cli.
	"cmd/broodforge": {row03 + "`cli` (the entry point; docs/design/08-implementation-plan.md P0)", []string{"internal/cli"}},

	// Test harnesses beside their component, imported by tests only (no component may import
	// one: none is in a component's entry).
	// 08 P2: clitest builds and runs the binary.
	"internal/cli/clitest": {"docs/design/08-implementation-plan.md P2 (clitest)", nil},
	// 08 P2: controltest serves control on a temporary socket.
	"internal/control/controltest": {"docs/design/08-implementation-plan.md P2 (controltest)", []string{"internal/control"}},
	// 05-debug-interface-git.md: gittest, a project and remote with the identity variables.
	"internal/git/gittest": {"docs/design/05-debug-interface-git.md (gittest)", nil},
	// 05-debug-interface.md: spawntest, the test binary standing in for the engine.
	"internal/spawn/spawntest": {"docs/design/05-debug-interface.md (spawntest)", nil},
	// 05-debug-interface.md: workflowtest makes the workflow tests' canonicals, with gittest.
	"internal/workflow/workflowtest": {"docs/design/05-debug-interface.md (workflowtest)", []string{"internal/git/gittest"}},

	// 08 P1: importfixture, a `go run` tool, not a subcommand.
	"internal/tools/importfixture": {"docs/design/08-implementation-plan.md P1 (importfixture)", nil},
	// tests/README.md, hook/linkworld: the trees the hook cases resolve against.
	"tests/hook/linkworld/canonicaltree": {"tests/README.md (hook/linkworld)", nil},
	"tests/hook/linkworld/sessiontree":   {"tests/README.md (hook/linkworld)", nil},
	// tests/README.md, lint/lint_test.go: this test.
	"tests/lint": {"tests/README.md (lint/lint_test.go)", nil},
}

// hostPackages may branch on the host anywhere: 03's "Host differences live in `spawn`,
// `worktree-remove`, `control` ..." and go.md's rule. The worktree removal spawn owns is
// internal/spawn/worktree.go, inside spawn.
var hostPackages = map[string]bool{
	"internal/spawn":   true,
	"internal/control": true,
}

// hostFiles are the only other files that may: setup's host files (go.md: "no other file of
// `setup` branches").
var hostFiles = map[string]bool{
	"internal/setup/host_windows.go": true,
	"internal/setup/host_linux.go":   true,
}

// anyFiles are internal/engine's raw stream layer, as go.md states it: parse.go and its test's
// oracle, which decodes the same raw lines.
var anyFiles = map[string]bool{
	"internal/engine/parse.go":      true,
	"internal/engine/parse_test.go": true,
}

// knownOS are the GOOS values a file name suffix or build tag can name; "unix" is a build tag only.
var knownOS = set("aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js",
	"linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos")

var knownArch = set("386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64", "mips",
	"mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc", "ppc64", "ppc64le", "riscv",
	"riscv64", "s390", "s390x", "sparc", "sparc64", "wasm")

func set(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// listed is the part of `go list -json` the test reads.
type listed struct {
	ImportPath     string
	Dir            string
	GoFiles        []string
	CgoFiles       []string
	TestGoFiles    []string
	XTestGoFiles   []string
	IgnoredGoFiles []string
	Imports        []string
	Error          *struct{ Err string }
}

// source is one parsed .go file.
type source struct {
	pkg  string // module-relative package path
	file *ast.File
	head string // first line, carriage return trimmed
}

func TestGoRules(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	module := modulePath(t, root)
	goos := []string{"linux", "windows"}
	lists := map[string][]listed{}
	for _, g := range goos {
		lists[g] = goList(t, root, g)
	}
	fset := token.NewFileSet()
	files := parseAll(t, fset, root, module, lists)

	t.Run("spdx", func(t *testing.T) {
		for _, name := range sortedKeys(files) {
			if h := files[name].head; h != spdx {
				t.Errorf("%s:1: first line %q, want %q (.claude/rules/go.md)", name, h, spdx)
			}
		}
	})

	t.Run("host-branching", func(t *testing.T) {
		for _, name := range sortedKeys(files) {
			s := files[name]
			if hostPackages[s.pkg] || hostFiles[name] {
				continue
			}
			for _, w := range hostBranches(fset, name, s.file) {
				t.Errorf("%s: host branching outside spawn, control and setup's host files (.claude/rules/go.md; 03-architecture.md, Components)", w)
			}
		}
		for f := range hostFiles {
			if _, ok := files[f]; !ok {
				t.Errorf("hostFiles names %s, which no package lists: remove it from tests/lint/lint_test.go", f)
			}
		}
	})

	t.Run("any", func(t *testing.T) {
		for _, name := range sortedKeys(files) {
			if anyFiles[name] {
				continue
			}
			for _, w := range anyUses(fset, files[name].file) {
				t.Errorf("%s:%s outside internal/engine's raw stream layer (.claude/rules/go.md)", name, w)
			}
		}
		for f := range anyFiles {
			if _, ok := files[f]; !ok {
				t.Errorf("anyFiles names %s, which no package lists: remove it from tests/lint/lint_test.go", f)
			}
		}
	})

	t.Run("imports", func(t *testing.T) {
		sites := importSites(fset, module, files)
		seen := map[string]bool{}
		var msgs []string
		add := func(m string) {
			if !seen[m] {
				seen[m] = true
				msgs = append(msgs, m)
			}
		}
		present := map[string]bool{}
		for _, g := range goos {
			for _, p := range lists[g] {
				pkg := strings.TrimPrefix(p.ImportPath, module+"/")
				present[pkg] = true
				if p.Error != nil {
					add(fmt.Sprintf("%s (GOOS=%s): go list: %s", pkg, g, p.Error.Err))
				}
				e, ok := graph[pkg]
				if !ok {
					add(fmt.Sprintf("package %s is not in the import table: add its row to docs/design/03-architecture.md (Components), or name it in the design if it is a harness, tool or fixture, and add its entry to graph in tests/lint/lint_test.go", pkg))
					continue
				}
				allowed := set(e.imports...)
				for _, imp := range p.Imports {
					to, ok := strings.CutPrefix(imp, module+"/")
					if !ok {
						continue
					}
					at := sites[pkg][to]
					if at == "" {
						at = pkg
					}
					if _, known := graph[to]; !known {
						add(fmt.Sprintf("%s: %s imports %s, a package the import table does not know: add its row to docs/design/03-architecture.md (Components), or name it in the design if it is a harness, tool or fixture, and add its entry to graph in tests/lint/lint_test.go", at, pkg, to))
						continue
					}
					if !allowed[to] {
						add(fmt.Sprintf("%s: %s imports %s, which %s does not name: edit that row and its entry in tests/lint/lint_test.go, or drop the import", at, pkg, to, e.row))
					}
				}
			}
		}
		for pkg := range graph {
			if !present[pkg] {
				add(fmt.Sprintf("graph names %s, which go list does not: remove its entry from tests/lint/lint_test.go", pkg))
			}
		}
		sort.Strings(msgs)
		for _, m := range msgs {
			t.Error(m)
		}
	})
}

// TestDetectors probes the file detectors on their own, with sources parsed here, so a
// detector that stops seeing a case fails even while the tree is clean.
func TestDetectors(t *testing.T) {
	parse := func(t *testing.T, fset *token.FileSet, name, src string) *ast.File {
		t.Helper()
		f, err := parser.ParseFile(fset, name, src, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}

	t.Run("anyUses", func(t *testing.T) {
		for _, c := range []struct {
			src  string
			want []string // "line: what"; nil for none
		}{
			{"package p\nvar a map[string]any\n", []string{"2: any"}},
			{"package p\nvar b interface{}\n", []string{"2: interface{}"}},
			{"package p\nvar c = make([]any, 0)\n", []string{"2: any"}},
			{"package p\nvar d = make(map[string]any)\n", []string{"2: any"}},
			{"package p\nvar e = new(any)\n", []string{"2: any"}},
			{"package p\nvar f = []any(nil)\n", []string{"2: any"}},
			{"package p\nfunc g(v int) {\n\tswitch v := interface{}(v).(type) {\n\tcase any:\n\t\t_ = v\n\t}\n}\n", []string{"3: interface{}", "4: any"}},
			{"package p\nfunc h[T any](x T) {}\nvar _ = h[any]\n", []string{"2: any", "3: any"}},
			{"package p\nfunc i(xs ...any) {}\n", []string{"2: any"}},
			{"package p\nvar j = any(1)\n", []string{"2: any"}},
			{"package p\ntype k interface{ ~int | ~string }\n", nil},
			{"package p\ntype q struct{ any bool }\nfunc (r q) m() bool { return r.any }\n", nil},
			{"package p\ntype q struct{ any bool }\nvar l = q{any: true}\n", nil},
			{"package p\nfunc any() {}\n", nil},
			{"package p\nfunc n(any int) {\n\tfor any := range 3 {\n\t\t_ = 0\n\t}\n}\n", nil},
		} {
			fset := token.NewFileSet()
			got := anyUses(fset, parse(t, fset, "p.go", c.src))
			if strings.Join(got, "|") != strings.Join(c.want, "|") {
				t.Errorf("anyUses(%q) = %q, want %q", c.src, got, c.want)
			}
		}
	})

	t.Run("hostBranches", func(t *testing.T) {
		for _, c := range []struct {
			name, src string
			want      int
		}{
			{"a.go", "package p\nimport \"runtime\"\nvar a = runtime.GOOS\n", 1},
			{"b.go", "package p\nimport rt \"runtime\"\nvar b = rt.GOOS\n", 1},
			{"c.go", "//go:build !windows\n\npackage p\n", 1},
			{"d.go", "//go:build unix\n\npackage p\n", 1},
			{"e.go", "// +build linux\n\npackage p\n", 1},
			{"f_windows.go", "package p\n", 1},
			{"g_linux.go", "//go:build linux\n\npackage p\nimport \"runtime\"\nvar g = runtime.GOOS\n", 3},
			{"h.go", "package p\nimport \"runtime\"\nvar h = runtime.GOARCH\n", 0},
			{"i.go", "//go:build go1.27 && !race\n\npackage p\n", 0},
			{"j.go", "package p\n\n//go:build windows\nvar j = 1\n", 0},
			{"k.go", "package p\ntype q struct{ GOOS string }\nvar k = q{}.GOOS\n", 0},
		} {
			fset := token.NewFileSet()
			if got := hostBranches(fset, c.name, parse(t, fset, c.name, c.src)); len(got) != c.want {
				t.Errorf("hostBranches(%s, %q) = %q, want %d", c.name, c.src, got, c.want)
			}
		}
	})

	t.Run("goosSuffix", func(t *testing.T) {
		for name, want := range map[string]bool{
			"x_windows.go": true, "x_linux_test.go": true, "x_linux_amd64.go": true, "x_darwin_arm64_test.go": true,
			"x.go": false, "windows.go": false, "x_unix.go": false, "x_amd64.go": false, "host_test.go": false,
			"x_windowsy.go": false,
		} {
			if got := goosSuffix(name); got != want {
				t.Errorf("goosSuffix(%s) = %v, want %v", name, got, want)
			}
		}
	})
}

// modulePath reads the module line of go.mod at root.
func modulePath(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("the module root is two directories up from tests/lint: %v", err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[0] == "module" {
			return strings.Trim(f[1], `"`)
		}
	}
	t.Fatal("go.mod has no module line")
	return ""
}

// goList runs `go list -e -json ./...` at root for one GOOS, cgo off as on the build line.
func goList(t *testing.T, root, goos string) []listed {
	t.Helper()
	cmd := exec.Command("go", "list", "-e", "-json", "./...")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS="+goos, "CGO_ENABLED=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list (GOOS=%s): %v\n%s", goos, err, stderr.String())
	}
	var pkgs []listed
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p listed
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("go list (GOOS=%s): %v", goos, err)
		}
		pkgs = append(pkgs, p)
	}
	if len(pkgs) == 0 {
		t.Fatalf("go list (GOOS=%s) listed no package", goos)
	}
	return pkgs
}

// parseAll parses every .go file a listing names, test files and the other host's files
// included, keyed by module-relative slash path.
func parseAll(t *testing.T, fset *token.FileSet, root, module string, lists map[string][]listed) map[string]source {
	t.Helper()
	files := map[string]source{}
	for _, pkgs := range lists {
		for _, p := range pkgs {
			pkg := strings.TrimPrefix(p.ImportPath, module+"/")
			for _, group := range [][]string{p.GoFiles, p.CgoFiles, p.TestGoFiles, p.XTestGoFiles, p.IgnoredGoFiles} {
				for _, f := range group {
					abs := filepath.Join(p.Dir, f)
					rel, err := filepath.Rel(root, abs)
					if err != nil {
						t.Fatal(err)
					}
					name := filepath.ToSlash(rel)
					if _, done := files[name]; done || !strings.HasSuffix(f, ".go") {
						continue
					}
					src, err := os.ReadFile(abs)
					if err != nil {
						t.Fatal(err)
					}
					af, err := parser.ParseFile(fset, name, src, parser.ParseComments|parser.SkipObjectResolution)
					if err != nil {
						t.Errorf("%v", err)
						continue
					}
					head, _, _ := strings.Cut(string(src), "\n")
					files[name] = source{pkg: pkg, file: af, head: strings.TrimSuffix(head, "\r")}
				}
			}
		}
	}
	return files
}

// hostBranches lists a file's host branching: a GOOS file name suffix, a build constraint
// naming a GOOS or unix, a use of runtime.GOOS.
func hostBranches(fset *token.FileSet, name string, f *ast.File) []string {
	var out []string
	if goosSuffix(filepath.Base(name)) {
		out = append(out, name+":1: a GOOS file name suffix")
	}
	for _, g := range f.Comments {
		if g.End() >= f.Package {
			break
		}
		for _, c := range g.List {
			if !constraint.IsGoBuild(c.Text) && !constraint.IsPlusBuild(c.Text) {
				continue
			}
			x, err := constraint.Parse(c.Text)
			if err != nil {
				continue
			}
			var host []string
			x.Eval(func(tag string) bool {
				if knownOS[tag] || tag == "unix" {
					host = append(host, tag)
				}
				return true
			})
			if len(host) > 0 {
				out = append(out, fmt.Sprintf("%s:%d: build constraint on %s", name, fset.Position(c.Pos()).Line, strings.Join(host, ", ")))
			}
		}
	}
	rt := ""
	for _, im := range f.Imports {
		if p, _ := strconv.Unquote(im.Path.Value); p == "runtime" {
			rt = "runtime"
			if im.Name != nil {
				rt = im.Name.Name
			}
		}
	}
	if rt != "" {
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok && s.Sel.Name == "GOOS" {
				if id, ok := s.X.(*ast.Ident); ok && id.Name == rt {
					out = append(out, fmt.Sprintf("%s:%d: runtime.GOOS", name, fset.Position(s.Pos()).Line))
				}
			}
			return true
		})
	}
	return out
}

// goosSuffix reports whether a file name carries a GOOS constraint as go/build reads it:
// *_GOOS.go or *_GOOS_GOARCH.go, with or without _test.
func goosSuffix(base string) bool {
	s := strings.TrimSuffix(strings.TrimSuffix(base, ".go"), "_test")
	parts := strings.Split(s, "_")
	n := len(parts)
	if n >= 3 && knownArch[parts[n-1]] && knownOS[parts[n-2]] {
		return true
	}
	return n >= 2 && knownOS[parts[n-1]]
}

// anyUses lists the lines where a file writes an empty interface or the identifier any. Every
// any counts except a name being declared (field, parameter, variable, function, method, type,
// label, import), a selector's field or method name, and a composite literal's key, so a
// struct field named any and its uses as x.any are not ones.
func anyUses(fset *token.FileSet, f *ast.File) []string {
	exempt := map[*ast.Ident]bool{}
	names := func(ids ...*ast.Ident) {
		for _, id := range ids {
			if id != nil {
				exempt[id] = true
			}
		}
	}
	exprs := func(es ...ast.Expr) {
		for _, e := range es {
			if id, ok := e.(*ast.Ident); ok {
				exempt[id] = true
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			names(x.Sel)
		case *ast.Field:
			names(x.Names...)
		case *ast.ValueSpec:
			names(x.Names...)
		case *ast.TypeSpec:
			names(x.Name)
		case *ast.FuncDecl:
			names(x.Name)
		case *ast.ImportSpec:
			names(x.Name)
		case *ast.LabeledStmt:
			names(x.Label)
		case *ast.BranchStmt:
			names(x.Label)
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE {
				exprs(x.Lhs...)
			}
		case *ast.RangeStmt:
			if x.Tok == token.DEFINE {
				exprs(x.Key, x.Value)
			}
		case *ast.CompositeLit:
			for _, e := range x.Elts {
				if kv, ok := e.(*ast.KeyValueExpr); ok {
					exprs(kv.Key)
				}
			}
		}
		return true
	})
	lines := map[int]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.InterfaceType:
			if x.Methods == nil || len(x.Methods.List) == 0 {
				lines[fset.Position(x.Pos()).Line] = "interface{}"
			}
		case *ast.Ident:
			if x.Name == "any" && !exempt[x] {
				lines[fset.Position(x.Pos()).Line] = "any"
			}
		}
		return true
	})
	var out []string
	for _, l := range sortedInts(lines) {
		out = append(out, fmt.Sprintf("%d: %s", l, lines[l]))
	}
	return out
}

// importSites maps a package and an imported module-relative path to the file:line of its
// first non-test import, so an import failure points at the line to change.
func importSites(fset *token.FileSet, module string, files map[string]source) map[string]map[string]string {
	sites := map[string]map[string]string{}
	for _, name := range sortedKeys(files) {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		s := files[name]
		for _, im := range s.file.Imports {
			p, err := strconv.Unquote(im.Path.Value)
			if err != nil {
				continue
			}
			to, ok := strings.CutPrefix(p, module+"/")
			if !ok {
				continue
			}
			if sites[s.pkg] == nil {
				sites[s.pkg] = map[string]string{}
			}
			if _, done := sites[s.pkg][to]; !done {
				sites[s.pkg][to] = fmt.Sprintf("%s:%d", name, fset.Position(im.Pos()).Line)
			}
		}
	}
	return sites
}

func sortedKeys(m map[string]source) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedInts(m map[int]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}
