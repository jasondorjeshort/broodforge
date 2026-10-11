#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// Status line for Claude Code. Reads the JSON Claude pipes on stdin, prints one line,
// and writes the rate-limit numbers to ~/.claude/usage-state.json so the
// usage-gate hook (.claude/hooks/usage-gate.js) can read them.
//
// Fields used are documented at https://code.claude.com/docs/en/statusline
// rate_limits.* is present only on claude.ai Pro/Max and only after the first API response.
'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');

const STATE_FILE = path.join(os.homedir(), '.claude', 'usage-state.json');
const WARN = 50;   // policy: no new work at or above this
const STOP = 80;   // policy: stop everything at or above this

let raw = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => (raw += c));
process.stdin.on('end', () => {
  let d = {};
  try { d = JSON.parse(raw); } catch (_) { /* print what we can */ }

  const model = (d.model && d.model.display_name) || '?';
  const ctx = d.context_window && d.context_window.used_percentage;
  const rl = d.rate_limits || {};
  const fiveH = rl.five_hour && rl.five_hour.used_percentage;
  const sevenD = rl.seven_day && rl.seven_day.used_percentage;
  const cost = d.cost && d.cost.total_cost_usd;

  // Persist for the gate hook.
  try {
    fs.writeFileSync(STATE_FILE, JSON.stringify({
      updated_at: Math.floor(Date.now() / 1000),
      session_id: d.session_id || null,
      five_hour: rl.five_hour || null,
      seven_day: rl.seven_day || null,
    }));
  } catch (_) { /* status line must never fail */ }

  const pct = (v) => (typeof v === 'number' ? `${Math.round(v)}%` : 'n/a');
  const worst = Math.max(
    typeof fiveH === 'number' ? fiveH : 0,
    typeof sevenD === 'number' ? sevenD : 0,
  );
  const flag = worst >= STOP ? ' STOP' : worst >= WARN ? ' HOLD' : '';

  const parts = [
    model,
    `ctx ${pct(ctx)}`,
    `5h ${pct(fiveH)}`,
    `7d ${pct(sevenD)}${flag}`,
  ];
  // cost.total_cost_usd is Claude Code's API-list-price estimate for this session's tokens
  // (subagents included). On a subscription it is informational only; the meters are the limit.
  if (typeof cost === 'number') parts.push(`~$${cost.toFixed(2)} list`);
  process.stdout.write(parts.join(' | '));
});
