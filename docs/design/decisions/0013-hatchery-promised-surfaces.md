# 0013. Hatchery's promised surfaces and its reference fitness check

Date: 2026-10-16
Status: proposed

Record: no proposal; this ADR applies ADR 0012 to the base itself, as its critic review asked
(objection 6). `proposed` because three of the surfaces are files the owner writes by hand (the
policy file, grant files, idea blocks), which ADR 0012 says the owner confirms; the owner's answer
to "is this the list, and is anything missing that you edit and expect to keep working" turns the
status to `accepted` with no other edit. The reference check (below) is technical and does not
wait for the answer.

## Context

Three repositories run the base's hooks and skills, two of them on older bases that share
`~/.claude` with this one, and the update skill reads one line of a project's own ADRs. ADRs
0009, 0010 and 0011 were each forced by a format or a file that something else depended on and
nobody had named: the base line's meaning, the update mode it implies, the usage cache that older
siblings still read. Verified 2026-10-16: `policy.js` treats an unknown key as a malformed file
and loads base defaults with a note on stderr (its header, line 12; the open idea
`2026-10-13-malformed-policy-falls-back-silently.md`), so a renamed or removed key silently
replaces a project's whole policy; the hooks and status line require only `fs`, `os`, `path`,
`child_process` and relative modules; ADR 0006 names `HOOK_POLICY_FILE` as the one test seam.

## Decision

### The promise

An addition (a new key with a default that preserves old behaviour, a new optional field, a new
block kind that older readers may ignore) is free and needs no ADR. An incompatible change (a
removal, a rename, a changed type or meaning, a changed format of existing output, a stricter
parse) is a major choice under `CLAUDE.md`: proposal, critic, ADR, and the ADR says how each
sibling repository migrates. There are no version numbers; the ADR that changed a surface is its
version, and `git log` of the carrying file is the history. A file or line written against an
older base loads under a newer one with the same meaning.

### The surfaces

| Surface | Carried by | Readers | Record |
|---|---|---|---|
| The policy file: every key, its type, default and meaning | `.claude/hooks/policy.json`, schema in `policy.js` | the three hooks, the status line, every project's own file | ADR 0006 key table, ADR 0011 |
| Grant files: the file shape and the directory | `~/.claude/usage-gate-overrides/*.json`, written by the owner | `usage-gate.js` | ADR 0006, The HOLD override |
| The idea file grammar: frontmatter keys and the three closing blocks | `docs/ideas/*.md` | `triage`, every drain in the three repositories | ADR 0007, `docs/ideas/README.md` |
| The base line | `^Hatchery base: [0-9a-f]{7,40}` in a project's `docs/design/decisions/` | `update-from-hatchery`, the `adr` skill | ADRs 0004, 0009, 0010 |
| The usage cache: directory, file name, shape, the prune bound (never a file under a day old), the old `usage-state.json` left alone | `~/.claude/usage-state/<session_id>.json` | `statusline.js`, `usage-gate.js`, of every base version on the machine | ADR 0011, hook specs section 3 |
| The hook seams: `HOOK_POLICY_FILE`; `test-hooks.js --input <file> <hook>` with `--home` and `--config-dir`; the exit codes and fail modes | `.claude/hooks/*.js`, `test-hooks.js` | a project reproducing a hook bug; the harness | ADR 0006 Test seams and Fail modes |

Not promised: the status line's text; the wording of any deny; harness case ids and labels;
skill, agent and rule prose; `policy.examples/`; the `cswap list --json` shape (consumed, not
ours; ADR 0011); the findings, reference and operator frontmatter, trigger: the first program
that reads them. A promise on a surface a user touches by hand (the first three rows) is the
owner's; the rest are technical.

### The reference fitness check

In `.claude/hooks/test-hooks.js`, case group `F`, so the test command runs it and pusher check 8
fails a push on it. Config `.claude/hooks/fitness.json`, read only through the harness:

```
{ "deps": { "budget": 0, "paths": [".claude/hooks/*.js", ".claude/statusline.js"] }, "skip": [] }
```

Check `deps`: in every file the `paths` match, each `require('<m>')` literal whose `<m>` does
not start with `./` or `../`, with a `node:` prefix stripped, must be in
`require('module').builtinModules`; the number of distinct other modules is at most `budget`.
Detail on failure: `<file>:<line> requires <m>` per offender. The harness line is the contract's
line with the case id in front: `PASS F1 deps`, `SKIP F1 deps: <reason> (<idea>)`, or
`FAIL F1 deps: <detail>; from .claude/hooks/fitness.json deps.budget; lift: skip entry
(coordinator, one push, named in the commit message) or ADR amendment (architect)`. A `skip`
entry is `{ "check": "deps", "reason", "date", "idea" }`; a missing field or an idea path that
does not exist fails the check with `skip entry malformed` as the detail. Cases, each isolated in
the scratch tree like the post-edit cases: F1 the real config and tree pass; F2 a scratch config
over a scratch tree whose one file requires `left-pad` fails, and the line carries `lift:`; F3
the same with a skip entry naming a scratch idea file that exists prints `SKIP` and passes; F4
the same skip with the idea path absent fails with `skip entry malformed`. `--only F` selects
them. Built by a `coder` from this section; Node built-ins only, as the check itself requires.

### Deferred

- A compatibility check for the surfaces above (a fingerprint of the policy schema and the idea
  grammar), trigger: the first incompatible change proposed to either, which then lands with it.
- A migration note format for sibling repositories, trigger: the first incompatible change.

## Consequences

- An incompatible change to the policy schema, the grammar, the base line or the cache is a
  proposal from now on; a new key or field is an ordinary change. Nothing visible changes today.
- Costs: one config file and four harness cases to keep; a budget of zero that any future hook
  dependency must argue past by ADR, which is the point.
- The owner's answer is one word; until it comes the status is `proposed` and the promise is
  observed as if accepted, since every row restates what ADRs 0006 to 0011 already fixed.

Predictions (the trigger scan reads these; superseding this ADR records each outcome first):
- Prediction, check when the next sibling update runs: the sibling edits none of its policy
  file, grants or idea files to take the update.
- Prediction, check by 2027-04-16: every change to the policy schema since this ADR added a key
  with a preserving default, or has an ADR; none renamed or removed one without.
- Prediction, check by 2027-04-16: group `F` has never been skipped, or each skip has its idea
  file and an amendment that followed.
