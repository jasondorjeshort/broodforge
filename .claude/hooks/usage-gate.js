#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// PreToolUse hook for the Agent and Workflow tools, the only calls it sees.
// Denies launching a subagent or workflow when plan usage (the higher of the 5-hour and 7-day
// windows) is at or above usage.hold in .claude/hooks/policy.json, unless an override grant that
// applies to this call lifts it, and at or above usage.deny, where a deny line is set, without
// exception (null, the base, means none: HOLD is then the only line). Other tools are never gated.
// Usage comes from ~/.claude/usage-state.json, which .claude/statusline.js writes.
// Spec: docs/design/decisions/0002-hook-specs.md section 3; policy: ADR 0006.
//
// Fails open: a crash (exit 1, no onFailure in settings.json) lets the launch through, because a
// broken gate must not stop all delegation. Stale or missing data allows with a note on stderr,
// or denies when the policy sets usage.whenStale to "deny". A malformed policy falls back to the
// base defaults and says so on stderr.
'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');
const { load, usageVerdict, readGrants, judgeGrants, holdCovered } = require('./policy');

const STATE_FILE = path.join(os.homedir(), '.claude', 'usage-state.json');
// Override grants (read and judged in policy.js, shared with the status line) lift HOLD, never the
// deny line, until "until" on their meters, for one session, one project or the user. Format and
// matching: ADR 0006, The HOLD override.
const POLICY_DOC = 'docs/design/decisions/0006-hook-policy-file.md';

function readJson(file) {
  try { return JSON.parse(fs.readFileSync(file, 'utf8')); } catch (_) { return null; }
}
function deny(reason) {
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'deny', permissionDecisionReason: reason },
  }));
  process.exit(0);
}
const note = (msg) => process.stderr.write(`usage-gate: ${msg}\n`);

// session_id and cwd from the PreToolUse payload; unparseable stdin is an empty payload.
function readPayload(raw) {
  let p = null;
  try { p = JSON.parse(raw); } catch (_) { /* empty payload */ }
  if (!p || typeof p !== 'object') p = {};
  const str = (v) => (typeof v === 'string' && v !== '' ? v : null);
  return { sessionId: str(p.session_id), cwd: str(p.cwd) };
}

const scopeText = (s) => (s.kind === 'user' ? 'user' : `${s.kind} ${s.value}`);

// One clause per malformed or out-of-scope grant; every expired grant shares one clause, placed where
// the first one falls, so leftover files do not lengthen each deny. The format lives in the ADR.
function clause(name, c) {
  if (c.status === 'malformed') return `An override file exists but is malformed (${name}; see ADR 0006, The HOLD override).`;
  return `An override exists for ${c.scope.kind} ${c.scope.value}, not this one.`;
}

function main(raw) {
  const loaded = load();
  if (loaded.error) note(`${loaded.error}; using the base defaults.`);
  const u = loaded.policy.usage;
  const now = Math.floor(Date.now() / 1000);

  const state = readJson(STATE_FILE);
  const age = state && typeof state.updatedAt === 'number' ? now - state.updatedAt : null;
  if (age === null || age > u.staleSeconds) {
    const what = age === null ? 'no usage data (the status line has not written any)' : `usage data is ${Math.round(age / 60)} min old`;
    if (u.whenStale === 'deny') {
      deny(`Usage gate: ${what}, and the policy denies delegation without fresh data. Check usage with /usage in an interactive session; the status line refreshes the data while a session is open.`);
    }
    note(`${what}; allowing. Run /usage to confirm.`);
    return;
  }

  // A window whose reset time has passed has reset; its last reading no longer counts.
  const pct = (w) => (w && typeof w.used_percentage === 'number' && !(typeof w.resets_at === 'number' && w.resets_at <= now)
    ? w.used_percentage : null);
  const fiveH = pct(state.five_hour);
  const sevenD = pct(state.seven_day);
  const fmt = (v) => (v === null ? 'n/a' : `${Math.round(v)}%`);
  const detail = `5h ${fmt(fiveH)}, 7d ${fmt(sevenD)}`;
  // Each meter is judged by its own thresholds (usage.fiveHour / usage.sevenDay, else the shared
  // hold/deny) and the worse verdict wins; the reason names the meter that tripped.
  const v = usageVerdict(u, { '5h': fiveH, '7d': sevenD });
  const tripped = v.hits.map((h) => `${h.label} at or above ${h.threshold}%`).join(', ');

  if (v.level === 2) deny(`Usage gate: ${detail}. ${tripped}: ${u.denyAdvice}`);
  if (v.level === 1) {
    if (u.overrideEnv && process.env[u.overrideEnv] === '1') {
      note(`HOLD (${detail}) lifted for this session by ${u.overrideEnv}=1.`);
      return;
    }
    const ctx = readPayload(raw);
    const { grants, applying, lifted } = judgeGrants(readGrants(), now, ctx);
    const labels = v.hits.map((h) => h.label);
    if (holdCovered(lifted, v)) {
      // The reason of every grant used, that is every applying grant that lifts a tripped meter.
      for (const g of applying.filter((a) => a.c.meters.some((m) => labels.includes(m)))) {
        note(`HOLD (${detail}) lifted on ${g.c.meters.join(' and ')} by override ${g.name} (${scopeText(g.c.scope)}) until ${new Date(g.grant.until * 1000).toLocaleString()}: ${typeof g.grant.reason === 'string' && g.grant.reason ? g.grant.reason : 'no reason given'}`);
      }
      return;
    }
    const partial = lifted.length ? ` The override in force lifts only ${lifted.join(' and ')}.` : '';
    const expired = grants.filter((g) => g.c.status === 'expired');
    const unused = grants.filter((g) => g.c.status !== 'applies').map((g) => {
      if (g.c.status !== 'expired') return ` ${clause(g.name, g.c)}`;
      return g === expired[0] ? ` Expired override files: ${expired.map((e) => e.name).join(', ')}.` : '';
    }).join('');
    deny(`Usage gate: ${detail}. ${tripped}: ${u.holdAdvice}${partial}${unused} Lifting this early is the user's decision (${POLICY_DOC}).`);
  }
}

let raw = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => (raw += c));
process.stdin.on('end', () => {
  main(raw);
  process.exit(0);
});
