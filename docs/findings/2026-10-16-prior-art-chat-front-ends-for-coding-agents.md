---
date: 2026-10-16
status: open
area: design / prior art
---

# Nothing shipped covers Broodforge's pillars together; build, and fold in the lessons below

## Question

The owner asked on 2026-10-16 whether tools like Broodforge already exist, whether to keep
building or adopt one, and which of their lessons the design should absorb. Design is complete
(`00` to `07`, ADRs 0020 to 0022), a spike is running, no application code exists, so switching
cost is at its lowest. Pillars compared: self-hosted Discord bot; drives the real Claude Code CLI
so each user's own subscription pays; permissions from Discord roles; a worktree per session with
human and model co-author attribution; permission prompts as buttons with a policy mode; per-user
config directory (ADR 0020); session isolation on Windows and Linux (ADR 0022); per-user token
tracking; an agent-drivable debug interface; one code base for the PC and the droplet; GPL.
Ruled out beforehand and not re-argued: a raw-API primary engine; a hosted import-any-repo SaaS.

## What was tried

Twelve web searches (standard and extended) and about thirty fetches on 2026-10-16: Anthropic's
docs (channels, channels reference, remote control, Slack, Claude Tag, cloud sessions,
self-hosted environments, legal and compliance, the Agent SDK support article), the official
Discord plugin's `README.md`, `ACCESS.md` and `server.ts`, the GitHub API record (licence,
created, last push, stars) of each open-source candidate, each candidate's README, three
`anthropics/claude-code` issues, and vendor docs for Devin, Copilot, Jules, Codex and OpenHands.
"Verified" below means a primary page was fetched this session; "snippet" means a search result
only. Not checked: Bun on Windows for the official plugins; OpenAB's and OpenHands's repository
metadata; Cursor's Slack trigger (snippet only); the April 2026 harness cut-off beyond press.

Primary pages fetched: `code.claude.com/docs/en/channels`, `.../channels-reference`,
`.../remote-control`, `.../slack`, `.../claude-code-on-the-web`, `.../self-hosted-environments`,
`.../legal-and-compliance`; `claude.com/docs/claude-tag/overview`;
`support.claude.com/en/articles/15036540-use-the-claude-agent-sdk-with-your-claude-plan`;
`github.com/anthropics/claude-plugins-official/tree/main/external_plugins/discord` (`README.md`,
`ACCESS.md`, `server.ts`); `github.com/anthropics/claude-code/issues/40266` and `/39782` (43625
read from a secondary copy, not fetched); `docs.openhands.dev/enterprise/integrations/slack`; `docs.devin.ai/integrations/slack`;
`docs.github.com/en/copilot/concepts/agents/coding-agent/about-coding-agent`; `jules.google/docs`;
`learn.chatgpt.com/docs/third-party/slack`; `docs.openclaw.ai/channels/discord`. Each open-source
row is `github.com/<owner>/<repo>` as named, plus `api.github.com/repos/<owner>/<repo>`.

## Result

### Anthropic's own routes (verified unless marked)

| Route | What it is | Covers | Does not cover |
|---|---|---|---|
| Claude Code channels, Discord plugin (research preview; plugin repo Apache-2.0, pushed 2026-10-16) | An MCP server pushes chat into one already-running interactive `claude`; sender allowlist by user id, guild channels opt-in per channel, permission relay as Allow/Deny buttons sent by DM to every allowed user; `-p` disables AskUserQuestion and plan approval; custom channels need `--dangerously-load-development-channels`, which `-p` and the SDK ignore | Discord, the owner's subscription, buttons | One login for everyone, one session per process, no roles, no worktree per session, no attribution, no usage per user, no debug surface |
| `claude remote-control` server mode (Pro/Max; API keys refused) | One process serves up to 32 sessions, `--spawn worktree` gives each its own git worktree; clients are claude.ai or the mobile app on the same account; "grants no one else access"; crashed sessions resume on the next message; forwarded non-permission dialogs expire after 5 min | Worktree per session, many sessions, one subscription, local host | Discord, other people, roles, attribution, debug surface |
| Claude Code in Slack (legacy; retired 2026-10-05 in Claude Tag workspaces) and Claude Tag (Team/Enterprise, public beta) | `@Claude` starts a cloud session; GitHub only; legacy bills the mentioning user's Pro/Max plan, Claude Tag bills the organisation under a per-channel spend limit and runs as a shared identity with no user account attached | Chat-started coding, PRs | Discord, self-hosting (self-hosted environments: Team/Enterprise runners, Anthropic control plane, GitHub only), per-user payment, attribution to the person |
| Agent SDK demos (not fetched; snippet only) | A React chat UI and others, "local development only" | Nothing of ours | No chat-platform example exists |

### Open-source chat front ends for coding agents

| Project | Verified facts | Covers | Lacks |
|---|---|---|---|
| ebibibi/ebi-agent-chat-relay (MIT, Python, since 2026-02-18, pushed 2026-10-16, 61 stars) | CLI subprocess per session with `--resume` and stream-json input; thread = session; Allow/Deny buttons, auto-deny after 2 min; AskUserQuestion and plan approval surfaced; `git worktree` per session (advisory, dirty trees never removed); subscriptions through "account pools" of pre-logged-in profile directories; localhost REST control plane with secret and host guard; SQLite; bwrap/container/SSH execution as preview; Linux in production, Windows "expected to work" | The closest match: CLI, worktrees, buttons, subscriptions, a control API | No roles (`allowed_user_ids`), one login pool rather than one per person, no attribution, no enforced reach, no Windows record |
| Guappa/ClaudeTalk (MIT, TS, since 2026-09-22, pushed 2026-10-12) | Agent SDK; channel = conversation; Approve once / Deny / Approve the rest of this turn; reads never asked; heuristic guard rules; mid-turn message injection; Windows, Linux, macOS; "no sandbox: a turn runs as you"; the host's subscription, "run one bridge per person" because sharing one subscription is not permitted | Buttons, resume, three hosts | Many users, roles, worktrees, attribution |
| raravel/Agent4Discord and the gooainemuon fork (no licence file in the API record; fork README says MIT; 2 and 0 stars, pushed 2026-10-01/02) | SDK `query()`; session = channel, tool calls in threads; buttons only for "risky" calls; `maxSessionsPerUser`; usage tracker | Streaming embeds, buttons | Roles, worktrees, attribution, per-user credentials |
| 10000DOO/discord-agent-bridge (MIT, Swift, macOS-first, pushed 2026-09-29) | Claude Code, Codex, Grok per channel; "Every server member is an admin": role tiers are stored and displayed but "do not restrict anything" | Multi-engine per channel | Enforcement; Linux and Windows secondary |
| t11z/claudecord (MIT, TS, 0 stars, pushed 2026-10-03) | Each member pastes `claude setup-token` output; "Every run is billed to them"; agentic mode off by default, Docker-first | Per-member billing intent | Stores pasted OAuth tokens, which the legal page forbids (below) |
| lyramakesmusic/claudebot-discord (no licence, Python, pushed 2026-10-12) | One persistent stream-json process per channel; `--dangerously-skip-permissions`; warns that long tasks disconnect | Streaming | Permissions, roles, worktrees |
| RichardAtCT/claude-code-telegram (README says MIT, API record null; Python, 2815 stars, pushed 2026-10-01) | SDK primary, CLI fallback; allowlist, directory sandbox, token-bucket rate limit, per-user USD cap, audit log, SQLite, session per user per project | Caps, audit | Telegram; roles; worktrees; attribution |
| hanxiao/claudecode-telegram (no licence, 610 stars, created and last pushed 2026-01-25) | tmux injection plus a Stop hook (snippet) | Nothing durable | Unmaintained |
| osisdie/claude-code-channels (MIT, pushed 2026-04-04) | Wrapper around the official plugins plus brokers for Slack, LINE, WhatsApp | Not a bot | Inherits every channel limit |
| openabdev/openab (MIT per README; Rust; stars unverified) | ACP over stdio JSON-RPC to Claude Code, Codex, Gemini, Kiro; Discord and Slack; one CLI process per thread; user and channel allowlists; Helm first; "permission auto-reply" | Multi-engine boundary | Roles, worktrees, attribution, permissions as prompts |
| openclaw/openclaw (MIT, TS, 392k stars, pushed 2026-10-16) | Raw-API personal assistant gateway with a Discord channel (guild allowlists, role routing, exec approvals as components); press reports Anthropic cut subscription access to it on 2026-04-04; CVE-2026-25253 on its exposed gateway port and 341 malicious ClawHub skills (secondary sources) | Discord roles, approvals | Violates "subscription pays"; not a worktree-per-session coding tool |
| OpenHands Slack (Enterprise self-hosted install, public HTTPS required) | `@openhands` in Slack; only the thread starter may follow up; the bot feeds about 21 recent messages into the system prompt; raw LLM keys | Self-hosted chat coding | Violates "subscription pays"; Slack only |

Hosted agents reachable from chat, all violating the self-hosted principle and "subscription pays"
(verified unless marked): Devin (`@Devin` in Slack, per-session "code channel" with status, linked
PRs and a live worklog, "Outposts" to run on your own environment); GitHub Copilot coding agent
(one repository, one branch, one PR per task, a hard 59-minute session cap, Teams and Slack through
integrations); Jules (web only, plan approved before any change); ChatGPT in Slack with Codex
Cloud delegation; Cursor background agents from Slack (snippet). Gemini CLI has no chat front
beyond OpenAB.

### Terms, re-checked at the source

The Claude Code legal page (fetched today) says a platform may host the unmodified binary when
"each end user must authenticate with their own ... Claude subscription plan credentials", and
that developers "may not collect, store, or intermediate Claude.ai credentials or session
tokens". It also says "Anthropic does not permit third-party developers to offer Claude.ai login
into their own applications, or to route requests through Free, Pro, or Max plan credentials on
behalf of their users" and "sign-in to a Claude account must complete through Anthropic's own
flow", limited by "Nor does it prevent an end user from signing in to the unmodified Claude Code
binary ... where a platform hosts Claude Code". The support article "Use the Claude Agent SDK with
your Claude plan" says (update of 2026-10-07) "You can still use the Claude Agent SDK, `claude -p`,
and third-party apps with your subscription limits", that the announced change was paused on
2026-06-15, and that Max and Team plans now carry monthly API credits covering `claude -p`. The
April 2026 cut-off for OpenClaw and "all third-party harnesses" is press and a quoted customer
email only; no Anthropic page found names it. Broodforge's shape (unmodified binary, one login
directory per person, no token read) fits the hosting sentence; a raw-API or pasted-token design
does not. Not decided, asked of Anthropic once the droplet has users: whether ADR 0020's DM login relay,
where the bot passes the sign-in link and the pasted code through, is "Anthropic's own flow". Decided
by the owner 2026-10-16: the relay is off by default and API keys are the droplet fallback (last ADR
0020 amendment).

### Answer

1. Build. Nothing surveyed covers more than four of the eleven pillars, and the four-pillar
   candidate (ebi-agent-chat-relay) has no roles, no per-person login, no attribution and no
   enforced reach, which are the pillars the Utopia tiers depend on. Building on top of one would
   mean adopting a Python or Swift code base to replace its core; adopting the official Discord
   plugin would mean one login, one session and no roles. Confidence high.
2. Lessons, each with the doc it touches and whether it changes or confirms the design:
   - L1 (change, `04-ux.md` trace lines and permission prompt; `03-architecture.md` renderer):
     mask credential-shaped tokens in everything posted outbound. Anthropic's relay redacts
     provider tokens in `description` and `input_preview` since v2.1.234; our trace line posts the
     first 80 characters of every Bash command and the prompt posts the full command.
   - L2 (change, ADR 0020 context and `00-vision.md` question 1; the relay question it raised is decided, see the last ADR 0020 amendment): record the support article
     and the legal page as the primary-source answer to "headless use metered separately": paused,
     `-p` still on plan limits, API credits on Max and Team.
   - L3 (change, ADR 0020 consequences and `07-spike-plan.md`): `-p` disables AskUserQuestion
     and plan approval, so an agent that wants a structured answer must ask in prose and end the
     turn; state it, and have the spike show what the stream carries when the tool is called.
   - L4 (change, owner's call, `04-ux.md` "nobody answering" and R-15): every neighbour times a
     pending prompt out shorter than the session (ebi-agent-chat-relay denies after 2 min; Remote Control expires
     other dialogs after 5 min). A deny lets the turn finish and keeps its output; our 60-minute
     idle timeout ends the session instead. Consider a per-channel prompt timeout that denies.
   - L5 (confirmation, `04-ux.md` permission prompt): the prompt shows the command or path.
     Issue 40266 called the official plugin's tool-name-only prompt "approving blindly".
   - L6 (confirmation, R-25 and `03-architecture.md`): prompts go to one venue and one answer
     list. The official plugin DMs every allowed user; issue 43625 reported relay to every channel.
   - L7 (confirmation, N-18 and the channels reference): gate on the sender's id, never the room.
   - L8 (confirmation, ADR 0020): a per-person login directory is the only compliant shape;
     claudecord's pasted `setup-token` is what the legal page forbids, and ClaudeTalk's own README
     says one bridge per person for the same reason.
   - L9 (confirmation, R-51 and `05-debug-interface.md`): discord-agent-bridge shipped roles that
     display and do not enforce; "start a session as a named user with named roles" is the test.
   - L10 (confirmation, ADR 0020 per-turn process): persistent processes disconnect on long
     tasks (claudebot-discord's README; issue 39782's allowlist state expiring after a long task).
   - L11 (confirmation, `03-architecture.md` no TCP listener; ADR 0020 `--setting-sources user`):
     OpenClaw's exposed gateway port and its skill marketplace are the two failures to avoid.
   - L12 (confirmation, R-13 and Owner decision 2): only session participants' messages reach
     the agent; the Slack page warns Claude "may follow directions from other messages", and
     OpenHands feeds channel history into the system prompt.
   - L13 (confirmation of R-46's shape): per-user USD caps (claude-code-telegram) and per-channel
     spend limits (Claude Tag) are the granularities that shipped; keep `later`, keep both axes.
   - L14 (note for `docs/ideas/`): ACP (OpenAB, `claude-agent-acp`) is a candidate boundary for
     the R-42/R-43 second harness; the first-version `Engine` interface stays bespoke.
3. Residual risk from Anthropic's own routes: moderate for the owner's single-user case, low for
   the community case. `claude remote-control --spawn worktree` plus the Discord plugin already
   gives one person worktree-per-session coding from Discord on their own PC, and both are moving
   fast (channels gained permission relay, masking and buttons between March and October). If
   channels gain guild multi-user sessions, the PC-only, owner-only slice of Broodforge becomes a
   convenience. What a channel plugin cannot do, because it runs inside one login: let each Discord
   person pay with their own subscription, attribute commits to the Discord person, take
   permissions from Discord roles, enforce reach per channel on the host, and answer an agent from
   a terminal without Discord. Those are the Utopia pillars, and they are the reason to build.

## Next

The coordinator decides whether L1 to L4 become edits (L1 to L3 architect or tester; L4 is an
owner UX decision), adds L14 to `docs/ideas/`, and resolves this finding by promoting the terms
paragraph to ADR 0020's amendments. Re-run the survey when channels leave research preview.
