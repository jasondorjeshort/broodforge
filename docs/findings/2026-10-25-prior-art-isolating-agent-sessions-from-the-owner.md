---
date: 2026-10-25
status: resolved (ADR 0027)
area: setup
promoted_to: docs/design/decisions/0027-setup-in-few-steps.md
---

# Runners and agent sandboxes keep jobs off the owner's account; on Windows that means a dedicated account

## Question

ADR 0022 runs the PC instance as a local standard account `broodforge`, so a session's shell
cannot read the owner's profile (`~/.claude/.credentials.json`, `settings.json`, checkouts). The
droplet uses two users and bubblewrap. Proposal 0027 weighed a single-account default on the PC to
save the account's setup steps (a second account, a one-time sign-in, a scheduled task with a
password); its critic and the coordinator would keep the account. The owner ("we don't want to
have claude credentials in the session history") asked for this to be checked against practice.
Asked per tool: does it run jobs as the invoking user or a dedicated identity by default; what it
says about a job reaching the owner's credentials and home; what the separation costs the owner;
whether a sandbox replaces the separate account on Windows and on Linux. Then: the prevailing
practice per OS, the setup cost the field accepts, and whether a Windows sandbox for the engine
exists today that meets ADR 0022's supersede trigger ("the engine's sandbox gains a mode that wraps
the engine itself").

## What was tried

About ten web searches and thirty fetches on 2026-10-25; nothing was installed and this machine was
not changed. "Verified" means a primary page or the tool's source was fetched; "secondary" means
search snippets or third-party write-ups only. Primary sources:

- GitHub Actions runner: docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions;
  raw.githubusercontent.com/actions/runner/main/src/Runner.Listener/Configuration/WindowsServiceControlManager.cs
  and `.../src/Misc/layoutbin/systemd.svc.sh.template`.
- Azure Pipelines agent: learn.microsoft.com/en-us/azure/devops/pipelines/agents/agents (Interactive vs. service)
  and `.../agents/windows-agent` (Information security, Unattended config).
- GitLab Runner: docs.gitlab.com/runner/install/windows/, /runner/security/, /runner/install/linux-repository/.
- Jenkins: jenkins.io/doc/book/security/controller-isolation/ and /doc/book/managing/nodes/.
- Buildkite agent: buildkite.com/docs/agent/v3/windows, /v3/ubuntu, /v3/securing.
- Claude Code: code.claude.com/docs/en/sandboxing, /sandbox-environments, /corporate-launcher;
  github.com/anthropics/sandbox-runtime (README, Windows (alpha)).
- OpenAI Codex: learn.chatgpt.com/docs/windows/windows-sandbox and /docs/agent-approvals-security
  (redirect targets of developers.openai.com/codex/windows and /codex/sandbox).
- GitHub Copilot CLI: docs.github.com/en/copilot/how-tos/cloud-and-local-sandboxes/using-local-sandboxing.
- OpenHands: docs.openhands.dev/openhands/usage/runtimes/overview and /runtimes/local.
- Cursor: cursor.com/docs/cloud-agent and /docs/cloud-agent/self-hosted.
- Docker Sandboxes: docs.docker.com/ai/sandboxes/, /sandboxes/security/, /sandboxes/install/.
- Microsoft: learn.microsoft.com/en-us/windows-server/identity/ad-ds/manage/understand-service-accounts;
  learn.microsoft.com/en-us/windows/security/application-security/application-isolation/windows-sandbox/;
  blogs.windows.com/windows/2026/10/07/microsoft-execution-containers-policy-driven-containment-for-ai-agents/;
  github.com/microsoft/mxc (README).

Not verified: the GitHub runner's Windows default is a source comment ("We use NetworkService as
default account for actions runner"); the helper returning it was not read. GitLab's and
Buildkite's default Windows service account (the pages do not name it). That LocalSystem can read
every user profile and NetworkService cannot (default profile ACLs; general Windows knowledge, not
fetched). That virtual accounts apply to client Windows (the page's "Applies to" list was not
seen). Codex's sandbox user names and whether its commands can read the profile (third-party only;
openai.com's engineering post returned 403). Whether srt's Windows alpha can wrap a long-running
`claude -p` or run from a task with no interactive session. Whether MXC process containers or
Docker Sandboxes run on Windows 11 Home (neither page names editions). Devin was not looked at.

## Result

### Per tool

| Tool | Default identity of a job | Owner's credentials and home | Owner's setup cost | Sandbox instead of an account? |
|---|---|---|---|---|
| GitHub Actions runner | Windows: NetworkService by default (source comment); Linux `svc.sh install` runs as `$SUDO_USER`, the invoking user, unless a user is named (template) | Self-hosted runners "can be persistently compromised by untrusted code"; "should almost never be used for public repositories"; "the amount of sensitive information in this environment should be kept to a minimum" (hardening) | Elevated `config.cmd` with a service prompt; no password for the default account | No; JIT runners ("at most one job") as mitigation, with a warning that reused hardware can leak |
| Azure Pipelines agent | Service recommended; "On Windows, we recommend that you use a service account like Network Service or Local Service. These accounts permissions are restricted and their passwords don't expire" (agents) | "a software product designed to execute code it downloads"; "best practice to have the identity running the agent be different from the identity with permissions to connect the agent" (windows-agent) | Elevated config; `--windowsLogonPassword` "not required for Group Managed Service Accounts and Windows built in accounts" | No |
| GitLab Runner | Windows: "Built-in System Account (recommended) or ... a user account"; Linux: "a `gitlab-runner` user is created" | Shell executor: "jobs are run with the permissions of the GitLab Runner's user"; "use it only for running trusted builds" (security) | A user account needs `install --user ... --password ...`, "required to start the service by Windows" | Docker or VM executors for untrusted jobs |
| Jenkins | Builds on the built-in node "have the same level of access to the controller file system as the Jenkins process"; "highly advisable to not run any builds on the built-in node" | On one machine, run the agent as a separate OS user with "no file system access ... to the Jenkins home directory" and no `sudo` (controller isolation) | Task Scheduler with a chosen user and "Run whether user is logged on or not" (nodes) | No |
| Buildkite agent | Ubuntu: "runs as the `buildkite-agent` operating system user account"; Windows: "all commands run as the invoking user", account changed by `nssm set ... ObjectName` with a password | Untrusted commits: "contain the build scripts" in chroots, containers or VMs (securing) | Linux: none beyond the package; Windows: NSSM plus an account and password | Containers or VMs for untrusted code |
| Claude Code | The invoking user; the Bash sandbox is "macOS, Linux, and WSL2. On native Windows, Claude Code runs commands unsandboxed" | Sandbox reads "most of the machine, including credential files"; "no built-in credential deny list"; file tools, MCP servers and hooks run outside it | Linux: `bubblewrap`, `socat`; native Windows: none available | For the whole process: sandbox runtime (`srt claude`), dev container or VM; "Work on a native Windows host: A container or VM, or run the Bash sandbox inside WSL2"; `CLAUDE_CODE_PROCESS_WRAPPER` "On Windows ... is ignored" |
| Anthropic sandbox-runtime (srt) | Wraps "an entire process"; beta. Windows is "alpha": "The sandboxed process runs under a dedicated `srt-sandbox` local user account" | Windows default grants read on the profile to the sandbox group but denies `.ssh`, `.aws`, `.claude`, `.git-credentials`, browser data; `denyRead: ['~']` hides it all | Windows: `windows-install`, "one UAC prompt", creates the account and group, rotates the account's password on rerun, "No logout is required" | Yes on macOS and Linux; on Windows the sandbox is itself a second account. Launch is `CreateProcessWithLogonW` from the broker; Schannel clients (`curl.exe`, `Invoke-WebRequest`) fail inside |
| OpenAI Codex | Commands sandboxed by default: Seatbelt, "`bwrap` plus `seccomp`" on Linux; Windows `mxc` recommended, `elevated` and `unelevated` legacy | `elevated` "uses dedicated lower-privilege sandbox users"; under `workspace-write` `.git`, `.codex`, `.aws` stay read-only; profile reads not stated | `elevated`: UAC prompt, local users and groups, firewall rules, logon rights; `mxc`: "without administrator-approved setup, additional Windows accounts, or local firewall rules", Windows 11 24H2/25H2 builds `.9278` | Commands only, not the agent process (not stated otherwise) |
| GitHub Copilot CLI | Most commands and tools in an OS sandbox; Windows needs 25H2 with KB5124010 or 26H1 with KB5124006 | Writes cwd and temp; "This does not grant access to your entire home directory" | Opt-in `/sandbox enable` | Commands, not the CLI process |
| OpenHands | Docker sandbox default ("Good isolation from your host machine") | Process sandbox: "provides no sandbox isolation"; "can read/write files your user account can access" | Docker | Container |
| Cursor agents | Cloud: "isolated VMs in the cloud"; self-hosted worker runs "as the worker's OS user" | Secrets injected per workspace in the cloud; self-hosted page gives no isolation guidance | Cloud: none; self-hosted: a process manager | VM (cloud) |
| Docker Sandboxes | "The primary trust boundary is the microVM"; Anthropic lists it as a way to run Claude Code | Nothing outside the workspace and a skills store; "The raw credential values never enter the VM" (a host proxy injects them) | Windows 11, Windows Hypervisor Platform; `winget install -h Docker.sbx` "without administrator privileges" | Yes, a VM; Windows edition support not stated |

### Windows-only mechanisms

- Service identities (Microsoft): virtual accounts (`NT SERVICE\<name>`) are "managed local accounts"
  needing "No password management"; gMSA and dMSA are domain accounts. A standard local account is
  the "traditional" kind whose password "an administrator sets and rotates".
- Windows Sandbox: a disposable hypervisor VM; "currently not supported on Windows Home edition";
  "doesn't allow multiple instances to run simultaneously". The PC reports Windows 11 Home, build
  26300, so it is out on two counts.
- Microsoft Execution Containers (MXC), GA 2026-10-07 (Windows blog): can contain "the entire
  agent"; process containers on Windows 11, macOS, Linux; session containers run "under a distinct
  Windows account and session"; SDKs in Rust, .NET and Node plus a JSON executor `wxc-exec.exe`
  (README). Supported by Codex and Copilot; Claude Code is among those "releasing support", not yet
  shipped. An open Claude Code request asks for it (claudeissues.com/issue/69495, secondary).

### Conclusions

Windows. No runner defaults to the interactive owner. CI runners default to a built-in service
identity: NetworkService for GitHub and Azure, which Azure recommends over named accounts because
"passwords don't expire". GitLab recommends LocalSystem, which keeps jobs off the desktop but not out
of the owner's profile (inference, unverified). Every AI-agent tool that built a native Windows
boundary landed on a separate account: Codex `elevated`, Anthropic's own srt alpha and MXC session
containers. The single exception is MXC process containers, which are new, have no Claude Code
support and no stated edition list. ADR 0022's dedicated standard account matches this practice.
What differs is who pays for it. The field creates the account in its own setup, behind one UAC
prompt, with a password the tool generates and keeps (srt rotates it on rerun). No one asks the
owner to create an account, invent a password or sign in to it.

Linux. A package-created system user is the norm (`gitlab-runner`, `buildkite-agent`); the GitHub
runner's `$SUDO_USER` default is the counterexample. Sandboxes (bwrap in Claude Code, Codex, srt
and MXC) sit on top of that user, not in place of it. The droplet's two users plus bwrap is the
conservative end of normal.

Setup cost the field accepts. One elevated step at install (a service prompt, a UAC dialog or a
feature enable). Passwords appear only when the owner picks a named user (GitLab, Buildkite NSSM,
Azure autologon). No tool surveyed asks for an interactive first sign-in to the job's account. ADR
0022's cost sits above that line, so the practice argues for automating the account's setup. It
does not argue for dropping the account.

Windows sandbox for the engine today: none that meets the trigger. Claude Code's sandbox skips
native Windows, and its process wrapper is ignored there. srt wraps a whole process but is alpha on
Windows. It is itself a second account, launched with `CreateProcessWithLogonW`, which ADR 0022
records as failing from a non-interactive service, and it breaks Schannel clients. MXC is GA but
Claude Code has not adopted it. Windows Sandbox excludes Home. Docker Sandboxes (a microVM, credentials
held outside) is the one shipped whole-engine boundary on Windows, but it is a third-party
runtime, not "the engine's sandbox", and Home support is unconfirmed.

The owner's concern. The account keeps the owner's own credentials out of every session. It does
not keep a session's own credential out of its Bash, since on Windows a session can read
`harness/<id>/.credentials.json` (ADR 0022, Windows paragraph). The field closes that gap in two
ways. One holds the credential outside the boundary: Docker Sandboxes' proxy, and Claude cloud
sessions' GitHub token "outside the sandbox". The other denies the credential directories inside
it, as srt's default deny of `.claude` does. Neither exists on the PC today.

#### Worth copying

1. Setup creates the dedicated account itself, with one UAC prompt and a tool-generated password
   it stores and rotates (srt `windows-install`, Codex `codex sandbox setup --elevated`). This cuts
   0027's account steps and keeps the boundary.
2. A service identity with no password to manage where it fits, chosen over a named account for
   that reason (Azure's NetworkService/LocalService advice; Microsoft virtual accounts). Whether
   a Claude login and profile work under one is open.
3. The credential lives outside what the job can read, and well-known credential paths are denied
   by name even inside (Docker Sandboxes' injecting proxy; srt's deny list with `.claude`).

## Next

Resolved 2026-10-26: ADR 0027 keeps the account and folds practice 1 into `setup host` (Decision
5); whether the account and the task can be made with no password typed is claim W6, spiked in
P20 of `../design/08-implementation-plan-setup.md`. ADR 0022's trigger (Claude Code ships MXC
support, or srt leaves alpha on Windows) stays in that ADR's "Supersede when".
