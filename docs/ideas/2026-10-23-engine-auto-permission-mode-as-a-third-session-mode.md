---
date: 2026-10-23
kind: feature
related: [2026-10-14-ux-choices-that-could-become-options.md, 2026-10-22-discord-layer-trace-for-debugging-the-live-bot.md]
---

# The engine's `auto` permission mode as a third session mode

Priority: the next wave (owner, 2026-10-23, in session s0008; first set medium the same day, while
starting to build Broodforge inside Broodforge): `auto` must be settable as the
default for new sessions (instance `defaults.mode` and a channel's `mode`) and switchable inside a
running session (the owner suggested `/session permissions auto` or similar; today's switch is
`/session mode`, so whether `auto` joins it or a new command is added is the owner's wording call).

A session has two permission modes today: `prompt` ("asks first": every call the reach table does
not settle goes to Discord as an Allow/Deny prompt) and `policy` ("acts freely": Broodforge answers
the prompts itself, inside the channel's reach). Both run the engine as `--permission-mode default`
(`internal/engine/claudecode.go`, `Args`; `internal/policy`, `ModePrompt` and `ModePolicy`;
`04-ux-session-controls.md`, Policy mode). The owner wants the middle: the engine's own `auto` mode, where the
engine's classifier reviews the actions and only the risky ones ask.

What exists. The pinned engine (2.1.309) lists `auto` among the `--permission-mode` choices, next to
`acceptEdits`, `bypassPermissions`, `dontAsk`, `manual` and `plan`, and has `--permission-prompts
host|none` for who answers prompts under `-p`. ADR 0020 set `auto` aside and recorded the trigger:
"the engine's `auto` as a third mode (a channel wants classifier-reviewed actions)". That trigger
has fired. Whether this account and plan can use `auto` in `-p` mode is not yet observed; that is
the first thing to find out (a finding, `finding` skill), on the PC with the real engine.

Shape to consider: a third value for the channel and session `mode` (config key, `/session mode`
choices, the session row, the footer wording), mapped to `--permission-mode auto` at the spawn.
Broodforge's own safety must not depend on it: the `PreToolUse` hook and the reach table still
decide first (`03-hook-contract.md`), so `auto` can only make the engine's remainder cheaper,
never widen reach. Whatever the classifier would have prompted for must still reach a human in
Discord through the same prompt path as `prompt` mode.

## Open

- A prompt in prompt mode currently ends any subagent whose call it defers
  (`../findings/2026-10-23-subagents-in-a-session-end-before-writing.md`); the classifier's prompts
  under `auto` would take the same path, so `auto` inherits that problem until it is fixed.
- Does `auto` work under `-p --input-format stream-json` here, and what does the stream show when
  the classifier defers or refuses? (Probe on the PC first; the droplet row follows P14.)
- Where the hook and the classifier disagree: the hook's deny is final (C39); does an `auto`
  allow ever run before a hook answer? Test it in isolation, do not assume.
- Names people see (04, Owner 7): a third label next to `asks first` and `acts freely`; the
  owner's call, as is whether `auto` may be a channel default or only a session switch.
- Whether this is a major choice for the proposal loop (an incompatible change to the `mode`
  values in configuration and the debug surface); likely yes, so triage routes it to the architect.

2026-10-31, the owner: "Auto permission mode is a priority", with "the option to set this default
for a server or channel". The spike below runs now, at the front of the wave. On a pass, the third
value goes straight to an `architect` pass (ADR 0020 amendment, the `04-ux.md` header word and mode
rows, a third value of `defaults.mode` and of `Channel.Mode`, both already in `internal/config/types.go`;
additive, so not an incompatible change under ADR 0012, which is why no separate proposal) and then a coder pass; no separate proposal.

2026-10-31, spike run: `docs/findings/2026-10-31-spike-auto-permission-mode-starts-under-p-hook-still-blocks-classifier-never-fired.md` (starts under `-p`; the hook still blocks; the classifier's own defer or refuse not yet seen).

Plan: `coder`, one pass, P39 of `docs/design/08-implementation-plan-auto-mode.md`; the design is settled (ADR 0020
Amendments 2026-11-01, `03-hook-contract.md` Auto mode, `04-ux-session-controls.md`), and this idea is its source, not a
second brief. Files and Verify: P39's own, its Files list and its Proof (including `hook cases`, the repro and the
owner-gated real turn), not restated here. First step, before any file: C40a (a `Write` under `-p --permission-mode
default` with a hook exiting 0 silently), on the owner's word; an allow stops the pass at the finding. Out of scope: the
words people see (placeholders stay until the owner words them, 04 Owner 7); a relay of the classifier's refusal as a
venue prompt (the owner accepts or rejects shape B first). Delete this file in the commit that lands P39; the Open items
above are P39's or the finding's.
