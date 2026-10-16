#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// Status line for Claude Code. Reads the JSON Claude pipes on stdin, prints the working directory (when known)
// and then the meters line, and keeps this session's usage file, ~/.claude/usage-state/<session_id>.json,
// current for the usage-gate hook (.claude/hooks/usage-gate.js).
// Thresholds for the HOLD and STOP flags come from .claude/hooks/policy.json (usage block) and are
// shown per meter, "(hold 50)" or "(hold 50 stop 90)"; STOP shows only on a meter with a deny line,
// which the base does not set. "HOLD lifted" replaces HOLD when an override lifts it for this session.
// "acct <local part>" names the logged-in account (policy.readAccount) when it is known.
// Spec: docs/design/decisions/0002-hook-specs.md section 3; the per-session file: ADR 0011.
//
// Fields: https://code.claude.com/docs/en/statusline. rate_limits appears only on claude.ai
// plans and only after the first API response of a session, and each window (five_hour,
// seven_day) may be absent on its own, dropped once its resets_at passes. So the cache is merged
// per window: a window present on stdin replaces the cached one, an absent window keeps the cached
// one, and nothing is written when neither is present. A cached reading is never replaced by null.
// The account is a label, never the key: when the cached and current labels are both known and
// differ, cached windows absent from the payload are dropped before the merge.
// File: { account: { uuid, organizationUuid, email } | null, five_hour: { used_percentage, resets_at },
// seven_day: {...}, updatedAt } (epoch seconds). No session_id: nothing is read or written.
'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');

const STATE_DIR = path.join(os.homedir(), '.claude', 'usage-state');
const OLD_FILE = path.join(os.homedir(), '.claude', 'usage-state.json');
const WINDOWS = ['five_hour', 'seven_day'];
const PRUNE_SECONDS = 86400;
// A session id becomes a file name, so only plain ids are used; anything else counts as absent.
const SESSION_ID = /^[A-Za-z0-9_-]+$/;

function readState(file) {
  try {
    const j = JSON.parse(fs.readFileSync(file, 'utf8'));
    return j && typeof j === 'object' && !Array.isArray(j) ? j : {};
  } catch (_) { return {}; }
}

// Files in the directory older than a day, other than this session's, are deleted; so is the old
// single cache file, which nothing reads any more. Run when a session's file is created.
function prune(keep) {
  try { fs.rmSync(OLD_FILE, { force: true }); } catch (_) { /* a directory or locked: leave it */ }
  try {
    const cutoff = Date.now() - PRUNE_SECONDS * 1000;
    for (const name of fs.readdirSync(STATE_DIR)) {
      const f = path.join(STATE_DIR, name);
      if (f === keep || !(name.endsWith('.json') || name.endsWith('.tmp'))) continue;
      try {
        const st = fs.statSync(f);
        if (st.isFile() && st.mtimeMs < cutoff) fs.rmSync(f, { force: true });
      } catch (_) { /* vanished: nothing to do */ }
    }
  } catch (_) { /* no directory */ }
}

// Merges the windows present in rl over the cached state and writes it when one was present.
// current is the logged-in account's label or null; differ(a, b) is policy.accountsDiffer, or a
// function returning false when policy.js did not load. Rule order (ADR 0011): no window on stdin,
// nothing is written; else a label change drops the cached windows absent from this payload, then
// the merge; the current label is written when known, else the cached one is kept.
// The write goes to a temporary file and is renamed, so the gate never reads half a file.
function update(file, rl, current, differ) {
  const state = file ? readState(file) : {};
  const present = WINDOWS.filter((w) => rl && rl[w] && typeof rl[w].used_percentage === 'number');
  if (!present.length) return state;
  const out = { account: current || (state.account && typeof state.account === 'object' ? state.account : null) };
  const keep = !differ(state.account, current);
  for (const w of WINDOWS) {
    const v = rl[w];
    if (present.includes(w)) {
      out[w] = { used_percentage: v.used_percentage };
      if (typeof v.resets_at === 'number') out[w].resets_at = v.resets_at;
    } else if (keep && state[w]) out[w] = state[w];
  }
  out.updatedAt = Math.floor(Date.now() / 1000);
  if (!file) return out;
  const created = !fs.existsSync(file);
  const tmp = `${file}.${process.pid}.tmp`;
  try {
    fs.mkdirSync(STATE_DIR, { recursive: true });
    fs.writeFileSync(tmp, JSON.stringify(out));
    fs.renameSync(tmp, file);
    if (created) prune(file);
  } catch (_) {
    // The status line must never fail; do not leave a temporary file behind either.
    try { fs.rmSync(tmp, { force: true }); } catch (__) { /* nothing more to do */ }
  }
  return out;
}

let raw = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => (raw += c));
process.stdin.on('end', () => {
  let d = {};
  try { d = JSON.parse(raw) || {}; } catch (_) { /* print what we can */ }
  let usage = null;
  let policy = null;
  try { policy = require('./hooks/policy'); usage = policy.load().policy.usage; } catch (_) { usage = null; /* no flags */ }

  const sid = typeof d.session_id === 'string' && SESSION_ID.test(d.session_id) ? d.session_id : null;
  const file = sid ? path.join(STATE_DIR, `${sid}.json`) : null;
  const current = policy && typeof policy.readAccount === 'function' ? policy.readAccount() : null;
  const differ = (a, b) => { try { return !!policy && policy.accountsDiffer(a, b); } catch (_) { return false; } };
  // Without a session id there is no file: the meters show this payload's windows only.
  const state = update(file, d.rate_limits, current, differ);
  // The file's numbers belong to another account than the one logged in: show n/a, as the gate
  // treats them (stale).
  const otherAccount = differ(state.account, current);
  const now = Math.floor(Date.now() / 1000);
  const live = (w) => (!otherAccount && w && typeof w.used_percentage === 'number' && !(typeof w.resets_at === 'number' && w.resets_at <= now) ? w : null);
  const fiveH = live(state.five_hour);
  const sevenD = live(state.seven_day);
  const pct = (v) => (typeof v === 'number' ? `${Math.round(v)}%` : 'n/a');
  // Same judgement as the gate: each meter against its own thresholds, the worse one wins.
  const verdict = usage ? policy.usageVerdict(usage, { '5h': fiveH ? fiveH.used_percentage : null, '7d': sevenD ? sevenD.used_percentage : null }) : { level: 0, hits: [] };
  // "HOLD lifted" exactly when the gate would let a launch through (policy.holdLifted); any error
  // reading the grants just means not lifted.
  let lifted = false;
  if (verdict.level === 1) {
    const str = (v) => (typeof v === 'string' && v !== '' ? v : null);
    const ctx = { sessionId: str(d.session_id), cwd: str((d.workspace && d.workspace.current_dir) || d.cwd) };
    try { lifted = policy.holdLifted(usage, verdict, ctx, now, process.env); } catch (_) { lifted = false; }
  }
  const flag = verdict.level === 2 ? ' STOP' : verdict.level === 1 ? (lifted ? ' HOLD lifted' : ' HOLD') : '';
  // Each meter's thresholds, "(hold 50)" or "(hold 50 stop 90)"; none when the policy did not load.
  const limits = (label) => {
    if (!usage) return '';
    const t = policy.meterThresholds(usage).find(([l]) => l === label)[1];
    return ` (hold ${t.hold}${t.deny !== null ? ` stop ${t.deny}` : ''})`;
  };

  // resets_at is epoch seconds. Show the clock time if the reset is within 24h, else the weekday.
  const resetAt = (w) => {
    if (!w || typeof w.resets_at !== 'number') return '';
    const t = new Date(w.resets_at * 1000);
    const hm = `${String(t.getHours()).padStart(2, '0')}:${String(t.getMinutes()).padStart(2, '0')}`;
    if ((t - Date.now()) / 3600000 < 24) return ` ->${hm}`;
    return ` ->${['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'][t.getDay()]} ${hm}`;
  };

  // The account segment: the logged-in label's email local part, else the file's; omitted when
  // neither is known.
  const label = current || (state.account && typeof state.account === 'object' ? state.account : null);
  const email = label && typeof label.email === 'string' ? label.email : '';
  const local = email.split('@')[0];

  const parts = [
    (d.model && d.model.display_name) || '?',
    `ctx ${pct(d.context_window && d.context_window.used_percentage)}`,
    ...(local ? [`acct ${local}`] : []),
    `5h ${pct(fiveH && fiveH.used_percentage)}${limits('5h')}${resetAt(fiveH)}`,
    `7d ${pct(sevenD && sevenD.used_percentage)}${limits('7d')}${resetAt(sevenD)}${flag}`,
  ];
  // cost.total_cost_usd (API list-price estimate) is deliberately not shown: on a subscription the
  // meters and their reset times are what matter. Run /usage for the figure.
  // Line 1 is the working directory as received, so a wrong-checkout mistake shows; omitted (no blank
  // line) when the payload carries neither workspace.current_dir nor the older cwd.
  const dir = [d.workspace && d.workspace.current_dir, d.cwd].find((v) => typeof v === 'string' && v !== '');
  process.stdout.write((dir ? dir + '\n' : '') + parts.join(' | '));
});
