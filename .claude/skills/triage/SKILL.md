---
name: triage
description: Read every untriaged file in docs/ideas/ against the code it names and write one block into each — a Plan: where the work is settled and pass-sized, a Question: with a blank Answer: where a choice is the owner's. Also runs the ADR 0012 scan (fired triggers, due predictions, fitness lines). Writes no code, deletes only an idea the code already has, pushes nothing.
---

# Triage

Reading an idea, deciding what should happen to it and making it happen are three jobs, and only
the first two are cheap. This skill does the first two, so that a later build pass takes only what
is ready and never has to judge. Run it when planning a wave, or when `docs/ideas/` has files with
no block. The format of a file and its three blocks is `docs/ideas/README.md`; read it first.

## Procedure

1. The scan. Read each `Deferred` item in `docs/design/decisions/` against the tree and list
   those whose trigger has fired; `grep -rn "^- Prediction, check" docs/design/decisions/ | grep -v " Outcome "` and list
   those past their date or whose trigger is met; run the test command and list `FAIL` and `SKIP`
   lines from fitness checks. Each hit becomes one idea file (`kind: chore`, no `related:`, named
   for the ADR and the trigger, an existing one gets a dated line instead) ending in `Plan:` that
   names the architect and the deliverable (a superseding ADR, an amendment, or, for a prediction,
   the Outcome recorded on its line and a finding if it failed). The report counts hits.
2. The carry-back queue (ADR 0019), in Hatchery only: where `git remote` lists `hatchery`, print
   `carry-back: skipped, a hatchery remote exists` and go on. Otherwise, in Bash from the root,
   read every `Carry-back` cell of the form `carry-back/<slug> at <sha7>` in the sibling checkouts
   beside this one (a directory whose `docs/design/decisions/` holds a base line; no sibling list
   is kept anywhere) and classify it by Hatchery's `main`:

   ```
   out=$(for f in $(grep -rlE "^Hatchery base: [0-9a-f]{7,40}" ../*/docs/design/decisions/); do
     d=$(echo "$f" | cut -d/ -f2)
     grep -E "^\|" "$f" | grep -oE "carry-back/[a-z0-9-]+ at [0-9a-f]{7,40}" | while read -r cell; do
       hit=$(git log main --format="%H %s" --grep="$cell" | grep -E "^[0-9a-f]{40} Carry-back( declined)?:" | head -1)
       case "$hit" in
         *" Carry-back declined:"*) s=declined ;;
         *" Carry-back:"*) s="landed ${hit:0:7}" ;;
         *) s=pending ;;
       esac
       echo "carry-back: $d $cell: $s"
     done
   done)
   [ -n "$out" ] && echo "$out"; echo "carry-back: $(printf '%s\n' "$out" | grep -c ': pending$') pending"
   ```

   The report carries every line and the count; the coordinator puts each pending one in the
   handoff's Next as a landing brief (`carry-back` skill, Land). A sibling not checked out beside
   Hatchery is not seen.
3. Run `node .claude/hooks/queue-list.js ideas --untriaged --json` and parse it: the files that
   end in no `Plan:`, `Question:` or `Waits:`, oldest first. Files the stderr names with a `?` were
   not read as untriaged; the report lists each with its field and value.
4. For each: read the file, then the anchor it names (the file, route, command or screen), then
   enough around it to know what the change would touch. An idea whose anchor no longer exists is
   a `Question:` ("the anchor moved to X; still wanted?"), not a deletion.
5. Decide which block it earns:
   - `Plan:` when the work is settled — nothing in it is a choice of the owner's — and fits one
     pass by one agent. Write the brief that pass will be handed: files to change, the shape of
     the change, how to verify, what is out of scope. Three to ten lines.
   - `Question:` when something is the owner's to decide: whether they want it, which of two
     shapes, whether it is too big for a pass and should become a design doc or a proposal. State
     the question in one sentence, give your recommendation with the reason in one or two more,
     and leave `Answer:` blank on its own line. "Not worth building" is always a Question with a
     recommendation, never a deletion.
   - Already in the code: say so in the commit message and delete the file. This is the one
     deletion triage makes, and it is a fact about the tree.
6. Append the block to the end of the file, below `## Open` if present. Keep the file under the
   doc cap. Do not rewrite the idea above it.
7. Commit each file alone, through whoever the policy lets commit (`.claude/hooks/policy.json`;
   in the base that is the `runner`, via the `commit` skill), with a one-line subject naming the
   idea and the block written; the body carries the reason a Question is the owner's. Do not push.
8. Report: how many scan hits, step 2's lines and its pending count, how many files read, how many got a Plan, how many a Question, which were deleted as
   already built, and anything you could not anchor.

## Bounds

- Triage writes blocks and nothing else: no code, no doc edits outside the idea file (except the
  scan's idea files and their rows where `docs/ideas/README.md` carries a table), no deletion for taste.
- A `Waits:` is the owner's to write. If a file already carries a `Waits:` or a `Question:` with
  an `Answer:`, it is not yours; leave it.
- One block per file. A file that needs both a Plan and a Question gets the Question; once the
  owner has answered, the next triage replaces that block with the Plan.
