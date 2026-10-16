# Brief for the critic (fill in and send as the Agent prompt)

Review proposal `docs/design/proposals/NNNN-<title>.md`. Read `CLAUDE.md`, the proposal, and the
design docs it cites, in that order.

Context the proposal does not state: <one or two sentences, or "none">.

Already ruled out and why (do not re-argue these): <list, or "nothing">.

Verify every item under "Claims to verify" against primary sources (docs, release notes, source)
and cite what you found. Then argue the strongest case against the recommendation and for the best
alternative. For every mechanism in the proposal that refuses, denies or fails closed, say who can
lift it, from where, while a session is running, and whether its refusal message says so; a refusal
with no documented lift is an objection. Then a pre-mortem: it is ten years on and this choice
is the one the project regrets; say in one paragraph what went wrong and which line of the
proposal should have caught it. A prediction that cannot fail, an exit cost or moving-parts line
that is boilerplate, is an objection. Produce a numbered list of objections with severity (blocking, serious, minor),
evidence, and what would resolve each; then one paragraph with your own recommendation and
confidence.

Write the review to `docs/design/proposals/NNNN-critic-review.md` using the Write tool (Bash
heredocs with apostrophes fail here). Do not edit the proposal. Report the file path and the
count of objections by severity.
