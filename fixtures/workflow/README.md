# fixtures/workflow/

A canonical tree for the workflow copy (ADR 0025 rules 5 and 6; `docs/design/05-debug-interface.md`,
The workflow copy). The tests copy it into a temporary directory, commit it on `main` and use that
repository as a scratch root's canonical (`scratch new --project <repo>`); this README is not
listed by the copy, which takes `CLAUDE.md` and `.claude` only.

What it holds: a `CLAUDE.md` with one `@` import that resolves and one that does not; two agents
(one with a folded description and a block list of tools); three rules, one path-scoped; a
`.claude/hooks/policy.json` whose `privacy.identities` lists one placeholder identity;
`git-guard.js` and `post-edit.js`, each printing one line; and a `settings.json` registering them
under `PreToolUse` and `PostToolUse`, with a `Stop` entry the copy never keeps.

Reproduce a case: copy this directory, edit it, commit it there, then

    broodforge scratch new --project <repo>
    broodforge workflow render --root <scratch root> --project scratch --out <tmp>
    broodforge workflow probe --root <scratch root> --project scratch

with `"workflow": {"hooks": ["PreToolUse", "PostToolUse"]}` added to the root's
`config/projects/scratch.json` for `hooks.json` and the probe.
