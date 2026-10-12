#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// PreToolUse hook for the Agent and Workflow tools.
// Denies launching new subagents when plan usage (max of 5-hour and 7-day) is at or
// above the HOLD threshold. Usage comes from ~/.claude/usage-state.json, which
// .claude/statusline.js writes every time the status line refreshes.
//
// Policy (see CLAUDE.md): <50% normal, 50-80% no new work, >=80% stop.
'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');

const STATE_FILE = path.join(os.homedir(), '.claude', 'usage-state.json');
// Override: written by the coordinator only on the user's explicit instruction.
// JSON {"until": <epoch seconds>, "reason": "<who asked and why>"}. Lifts HOLD (not STOP)
// until `until`; the hook reports the reason on stderr each time it is used.
const OVERRIDE_FILE = path.join(os.homedir(), '.claude', 'usage-gate-override.json');
const HOLD = 50;
const STOP = 80;
const STALE_SECONDS = 2 * 60 * 60; // treat data older than 2h as unknown

function activeOverride() {
  try {
    const o = JSON.parse(fs.readFileSync(OVERRIDE_FILE, 'utf8'));
    if (typeof o.until !== 'number' || o.until <= Math.floor(Date.now() / 1000)) return null;
    return o;
  } catch (_) { return null; }
}

function allow() { process.exit(0); }
function deny(reason) {
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: {
      hookEventName: 'PreToolUse',
      permissionDecision: 'deny',
      permissionDecisionReason: reason,
    },
  }));
  process.exit(0);
}

let raw = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => (raw += c));
process.stdin.on('end', () => {
  let state = null;
  try { state = JSON.parse(fs.readFileSync(STATE_FILE, 'utf8')); } catch (_) { /* no data */ }

  if (!state || !state.updated_at) {
    process.stderr.write('usage-gate: no usage data yet (status line has not run); allowing.\n');
    return allow();
  }
  const age = Math.floor(Date.now() / 1000) - state.updated_at;
  if (age > STALE_SECONDS) {
    process.stderr.write(`usage-gate: usage data is ${Math.round(age / 60)} min old; allowing. Run /usage to confirm.\n`);
    return allow();
  }

  const fiveH = state.five_hour && state.five_hour.used_percentage;
  const sevenD = state.seven_day && state.seven_day.used_percentage;
  const f = typeof fiveH === 'number' ? fiveH : 0;
  const s = typeof sevenD === 'number' ? sevenD : 0;
  const worst = Math.max(f, s);

  const fmt = (v) => (typeof v === 'number' ? `${Math.round(v)}%` : 'n/a');
  const detail = `5h ${fmt(fiveH)}, 7d ${fmt(sevenD)}`;

  if (worst >= STOP) {
    return deny(`Usage gate: ${detail}. At or above ${STOP}%: stop all work, run the handoff skill, commit WIP from the main session, end the turn.`);
  }
  if (worst >= HOLD) {
    const o = activeOverride();
    if (o) {
      process.stderr.write(`usage-gate: HOLD (${detail}) lifted by override until ${new Date(o.until * 1000).toLocaleString()}: ${o.reason || 'no reason given'}\n`);
      return allow();
    }
    return deny(`Usage gate: ${detail}. At or above ${HOLD}%: do not start new subagents. Let in-flight agents finish, run the handoff skill, commit from the main session, then report. On the user's explicit instruction only, write ~/.claude/usage-gate-override.json {"until": <epoch>, "reason": "..."} to lift HOLD.`);
  }
  allow();
});
