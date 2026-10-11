---
name: finding
description: Record the result of an investigation (root cause, benchmark, failed approach, environment quirk) in docs/findings/ with status, and promote it when resolved. Use when a root cause is found, when something did not work and the reason is known, after a benchmark or comparison, or when a tool behaves unexpectedly.
---

# Finding

Findings are the evidence trail. They are dated and status-tagged so a reader knows whether to
trust them. A finding is written when the investigation produces something worth not repeating,
even if the answer is "unresolved, here is what was ruled out".

## Record

1. File: `docs/findings/YYYY-MM-DD-<slug>.md`, written with the Write tool from `template.md` in
   this directory. Frontmatter: `date`, `status: open`, `area` (free text, e.g.
   `build / android`), optional `supersedes: <file>`.
2. Sections: Question (what was being asked), What was tried (commands, versions, environment),
   Result (what is known now, with evidence), Next (what would resolve it; omit when resolved).
3. Add a line to `docs/findings/README.md` with status and a one-line summary.
4. If status is open and the coordinator should act, add the path under Open findings in
   `docs/HANDOFF.md`.

## Resolve

1. Set `status: resolved` and fill Result with the answer. Delete Next or leave what remains.
2. Promote the durable part: a fact about a tool or environment goes to `docs/reference/<topic>.md`
   (create or extend; set `verified:` to today); a decision goes through the `adr` skill. Set
   `promoted_to: <path>` in the finding. The finding keeps the evidence; the reference keeps the fact.
3. Update the index line and remove the entry from `HANDOFF.md`.

## Supersede

When a later finding replaces an earlier one, write the new file with `supersedes: <old file>`,
set the old to `status: superseded`, and update both index lines.

## Rules

- Cap 150 lines; link to logs rather than pasting them. Large evidence goes under `tmp/`
  (gitignored) or in the report, not in the finding.
- Never edit a finding to say something different from what was observed; write a new one.
