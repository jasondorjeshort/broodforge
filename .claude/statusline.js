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

  // resets_at is epoch seconds. Show the clock time if the reset is within 24h, else the weekday.
  const resetAt = (epoch) => {
    if (typeof epoch !== 'number') return '';
    const t = new Date(epoch * 1000);
    const hours = (t - Date.now()) / 3600000;
    const hm = `${String(t.getHours()).padStart(2, '0')}:${String(t.getMinutes()).padStart(2, '0')}`;
    if (hours < 24) return ` ->${hm}`;
    const day = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'][t.getDay()];
    return ` ->${day} ${hm}`;
  };

  const parts = [
    model,
    `ctx ${pct(ctx)}`,
    `5h ${pct(fiveH)}${resetAt(rl.five_hour && rl.five_hour.resets_at)}`,
    `7d ${pct(sevenD)}${resetAt(rl.seven_day && rl.seven_day.resets_at)}${flag}`,
  ];
  // cost.total_cost_usd (API list-price estimate, subagents included) is deliberately not shown:
  // on a subscription the meters and their reset times are what matter. Run /usage for the figure.
  void cost;
  process.stdout.write(parts.join(' | '));
});
