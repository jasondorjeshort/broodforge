#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// PreToolUse hook for the Agent and Workflow tools, the only calls it sees.
// Denies launching a subagent or workflow when plan usage (the higher of the 5-hour and 7-day
// windows) is at or above usage.hold in .claude/hooks/policy.json, override file aside, and at or
// above usage.deny without exception. Other tools are never gated here.
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
const { load } = require('./policy');

const STATE_FILE = path.join(os.homedir(), '.claude', 'usage-state.json');
// Lifts HOLD (not the deny line) until `until`, epoch seconds; written only on the user's word.
const OVERRIDE_FILE = path.join(os.homedir(), '.claude', 'usage-gate-override.json');
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

function main() {
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
  const worst = Math.max(fiveH || 0, sevenD || 0);
  const fmt = (v) => (v === null ? 'n/a' : `${Math.round(v)}%`);
  const detail = `5h ${fmt(fiveH)}, 7d ${fmt(sevenD)}`;

  if (worst >= u.deny) deny(`Usage gate: ${detail}. At or above ${u.deny}%: ${u.denyAdvice}`);
  if (worst >= u.hold) {
    const o = readJson(OVERRIDE_FILE);
    if (o && typeof o.until === 'number' && o.until > now) {
      note(`HOLD (${detail}) lifted by override until ${new Date(o.until * 1000).toLocaleString()}: ${o.reason || 'no reason given'}`);
      return;
    }
    deny(`Usage gate: ${detail}. At or above ${u.hold}%: ${u.holdAdvice} Lifting this early is the user's decision (${POLICY_DOC}).`);
  }
}

let raw = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => (raw += c));
process.stdin.on('end', () => {
  main();
  process.exit(0);
});
