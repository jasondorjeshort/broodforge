// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/session"
	"github.com/jasondorjeshort/broodforge/internal/workflow"
)

// The workflow copy's commands (05-debug-interface.md, The workflow copy): ADR 0025's copy
// procedure and runtime probe against a project's canonical, without a session. Both read the
// configuration and the canonical only; render writes under --out, probe under a temporary
// directory it removes.

// workflowRender is `broodforge workflow render --project <id> --out <dir> (file)`: prints
// `workflow: <commit> <date>` or `workflow: none (...)`, then `wrote  <file>` per rendered file
// and `skipped  <file>  <why>` per file left out, exit 0.
var workflowRender = Command{
	Name:  "workflow render",
	Class: ClassFile,
	Flags: []Flag{workflowProjectFlag,
		{Name: "out", Kind: FlagString, Value: "<dir>", Usage: "an absent or empty directory the copy is made in"}},
	Local: runWorkflowRender,
}

// workflowProbe is `broodforge workflow probe --project <id> (file)`: makes the copy in a
// temporary directory and probes each hooks.json entry's runtime, `ok  <name>  <word>` each
// (--json: the entry with its runtime); the first failure prints the companion's not-started
// line and `error: workflow-probe-failed: <command> needs <word>`, exit 1.
var workflowProbe = Command{
	Name:  "workflow probe",
	Class: ClassFile,
	Flags: []Flag{workflowProjectFlag},
	Local: runWorkflowProbe,
}

var workflowProjectFlag = Flag{Name: "project", Kind: FlagString, Value: "<id>", Usage: "the project whose canonical is read"}

func init() {
	commands = append(commands, &workflowRender, &workflowProbe)
	required[&workflowRender] = []string{"project", "out"}
	required[&workflowProbe] = []string{"project"}
}

// The records the two commands print, and their text columns.
type (
	workflowLine struct {
		Workflow string  `json:"workflow"`
		Branch   string  `json:"branch"`
		Commit   *string `json:"commit"`
		Date     *string `json:"date"`
	}
	workflowFile struct {
		Action string `json:"action"` // wrote or skipped
		File   string `json:"file"`
		Why    string `json:"why,omitempty"`
	}
	probeRow struct {
		Status  string `json:"status"`
		Name    string `json:"name"`
		Runtime string `json:"runtime"`
	}
	notStartedLine struct {
		NotStarted string `json:"notStarted"`
	}
)

var (
	workflowLineColumns = []string{"workflow:"}
	wroteColumns        = []string{"action", "file"}
	skippedColumns      = []string{"action", "file", "why"}
	probeColumns        = []string{"status", "name", "runtime"}
	notStartedColumns   = []string{"notStarted"}
)

func encodeWorkflow[T workflowLine | workflowFile | probeRow | notStartedLine | workflow.Probed](v T) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}

// workflowOptions loads the configuration and returns the project's copy options.
func workflowOptions(env *Env) (string, workflow.Options, error) {
	c, err := config.Load(env.Root, credentialExists)
	if err != nil {
		return "", workflow.Options{}, configFailure(env, err)
	}
	id := env.String("project")
	p := c.Project(id)
	if p == nil {
		return "", workflow.Options{}, control.NewError(CodeUnknownProject, "no project "+id+" in "+c.Dir)
	}
	w := p.WorkflowOrDefault()
	return p.ID, workflow.Options{Canonical: p.Repo.Canonical, Branch: orDefault(p.Repo.DefaultBranch, "main"),
		Inject: w.Inject, Hooks: w.Hooks}, nil
}

// probeWorkflows is config check's runtime probe (ADR 0025 rule 6; the companion's The runtime
// probe): for each project that lists hook events and injects its repository, the copy is
// rendered into a temporary directory, removed after, and its hooks probed. A copy that cannot
// be made or a hook whose runtime is missing is a problem of the project's file, naming the
// command and the word.
func probeWorkflows(ctx context.Context, c *config.Config) ([]config.Problem, error) {
	var out []config.Problem
	for i := range c.Projects {
		p := &c.Projects[i]
		w := p.WorkflowOrDefault()
		if len(w.Hooks) == 0 || w.Inject == config.InjectNone {
			continue
		}
		file := "projects/" + p.ID + ".json"
		for _, f := range c.Files {
			if f.Name == "projects/"+p.ID+".json" {
				file = f.Path
			}
		}
		msg, err := probeProject(ctx, p, w)
		if err != nil {
			return nil, err
		}
		if msg != "" {
			out = append(out, config.Problem{File: file, Message: "workflow.hooks: " + msg})
		}
	}
	return out, nil
}

// probeProject is one project's probe: "" when every hook's runtime is there, else what fails.
func probeProject(ctx context.Context, p *config.Project, w config.Workflow) (msg string, err error) {
	tmp, err := os.MkdirTemp("", "broodforge-workflow-")
	if err != nil {
		return "", err
	}
	defer func() {
		if rerr := os.RemoveAll(tmp); err == nil {
			err = rerr
		}
	}()
	copied, err := workflow.Render(ctx, workflow.Options{Canonical: p.Repo.Canonical, Branch: orDefault(p.Repo.DefaultBranch, "main"),
		Inject: w.Inject, Hooks: w.Hooks}, tmp)
	var we *workflow.Error
	if errors.As(err, &we) {
		return control.OneLine(we.Error()), nil
	}
	if err != nil {
		return "", err
	}
	_, err = workflow.Probe(ctx, copied.Dir, copied.Hooks)
	var pf *workflow.ProbeFailure
	if errors.As(err, &pf) {
		return "the " + pf.Hook.Event + " hook " + control.OneLine(pf.Hook.Command) + " needs " + pf.Word + ", which is not installed here", nil
	}
	return "", err
}

// workflowError maps the package's refusals to the CLI's error line.
func workflowError(err error) error {
	var we *workflow.Error
	if errors.As(err, &we) {
		return control.NewError(we.Code, control.OneLine(we.Message))
	}
	return err
}

func runWorkflowRender(ctx context.Context, env *Env) error {
	_, o, err := workflowOptions(env)
	if err != nil {
		return err
	}
	c, err := workflow.Render(ctx, o, env.String("out"))
	if err != nil {
		return workflowError(err)
	}
	line := workflowLine{Workflow: c.Line(), Branch: c.Branch}
	if c.Commit != "" {
		commit, date := c.Commit, FormatTime(c.Date)
		line.Commit, line.Date = &commit, &date
	}
	raw, err := encodeWorkflow(line)
	if err != nil {
		return err
	}
	if err := env.Out.Record(raw, workflowLineColumns); err != nil {
		return err
	}
	for _, f := range c.Files {
		if err := printWorkflowFile(env, workflowFile{Action: "wrote", File: f}, wroteColumns); err != nil {
			return err
		}
	}
	for _, s := range c.Skips {
		if err := printWorkflowFile(env, workflowFile{Action: "skipped", File: s.File, Why: s.Why}, skippedColumns); err != nil {
			return err
		}
	}
	return nil
}

func printWorkflowFile(env *Env, f workflowFile, cols []string) error {
	raw, err := encodeWorkflow(f)
	if err != nil {
		return err
	}
	return env.Out.Record(raw, cols)
}

func runWorkflowProbe(ctx context.Context, env *Env) (err error) {
	id, o, err := workflowOptions(env)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "broodforge-workflow-")
	if err != nil {
		return err
	}
	defer func() {
		if rerr := os.RemoveAll(tmp); err == nil {
			err = rerr
		}
	}()
	c, err := workflow.Render(ctx, o, tmp)
	if err != nil {
		return workflowError(err)
	}
	probed, perr := workflow.Probe(ctx, c.Dir, c.Hooks)
	for _, p := range probed {
		var raw json.RawMessage
		var cols []string
		if env.JSON {
			raw, err = encodeWorkflow(p)
		} else {
			raw, err = encodeWorkflow(probeRow{"ok", p.Name, p.Runtime})
			cols = probeColumns
		}
		if err != nil {
			return err
		}
		if err := env.Out.Record(raw, cols); err != nil {
			return err
		}
	}
	var pf *workflow.ProbeFailure
	if errors.As(perr, &pf) {
		raw, err := encodeWorkflow(notStartedLine{pf.NotStarted(id)})
		if err != nil {
			return err
		}
		if err := env.Out.Record(raw, notStartedColumns); err != nil {
			return err
		}
		return control.NewError(workflow.CodeProbeFailed, control.OneLine(pf.Hook.Command+" needs "+pf.Word))
	}
	return perr
}

// sessionWorkflow is the session service's copy at start (ADR 0025 rules 5 and 6, the session
// half): the copy rendered into the session's directory under workflow/, then, when hooks.json
// has entries, the runtime probe where the children will run. A refusal removes the directory
// when this call made it, and is a *session.WorkflowRefusal; a probe failure is
// workflow-probe-failed: <command> needs <word>, which the starter alone sees.
func sessionWorkflow(ctx context.Context, req session.WorkflowRequest) (session.WorkflowCopy, error) {
	_, statErr := os.Stat(req.Dir)
	made := os.IsNotExist(statErr)
	refuse := func(err error) (session.WorkflowCopy, error) {
		if made {
			_ = os.RemoveAll(req.Dir)
		}
		var we *workflow.Error
		var pf *workflow.ProbeFailure
		switch {
		case errors.As(err, &pf):
			return session.WorkflowCopy{}, &session.WorkflowRefusal{Code: workflow.CodeProbeFailed,
				Message: control.OneLine(pf.Hook.Command + " needs " + pf.Word)}
		case errors.As(err, &we):
			return session.WorkflowCopy{}, &session.WorkflowRefusal{Code: we.Code, Message: control.OneLine(we.Message)}
		}
		return session.WorkflowCopy{}, err
	}
	c, err := workflow.Render(ctx, workflow.Options{Canonical: req.Canonical, Branch: req.Branch, Inject: req.Key.Inject,
		Hooks: req.Key.Hooks}, req.Dir)
	if err != nil {
		return refuse(err)
	}
	if len(c.Hooks) > 0 {
		if _, err := workflow.Probe(ctx, c.Dir, c.Hooks); err != nil {
			return refuse(err)
		}
	}
	out := session.WorkflowCopy{Line: c.Line(), Copied: !c.None(), Branch: c.Branch, Commit: c.Commit, Date: c.Date,
		Hooks: []string{}}
	if len(c.Agents) > 0 {
		out.Agents = c.Dir + "/" + workflow.AgentsFile
	}
	for _, h := range c.Hooks {
		out.Hooks = append(out.Hooks, h.Name)
	}
	return out, nil
}
