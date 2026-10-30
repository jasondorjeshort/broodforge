---
date: 2026-10-29
kind: chore
---
# ADR 0027 prediction: a `setup` rerun on a configured host changes nothing

ADR 0027 (`../design/decisions/0027-setup-in-few-steps.md`, Consequences) holds the line
"Prediction, check when P21 lands: a rerun of `setup` on a configured host with the bot stopped
prints only `ok` and `skip` lines and changes no file, no credential, no channel and no role."
P21 landed in `fab65e1` (`internal/setup/`), so the trigger is met and the line has no Outcome.

Plan: the `architect` records the Outcome on that prediction line in
`docs/design/decisions/0027-setup-in-few-steps.md`, after the check: run `broodforge setup` twice on
a configured scratch host with the bot stopped and compare the output lines and the files,
credentials, channels and roles between runs (the fake Discord of `internal/setup/fakediscord_test.go`
stands in for the guild). If it holds, the Outcome says so with the date; if it fails, the Outcome
names the step that wrote and a finding in `docs/findings/` records the cause. Out of scope: the
P23 prediction on the same ADR, which waits for P23.
