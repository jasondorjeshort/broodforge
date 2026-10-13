#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// Loads .claude/hooks/policy.json, the one file that holds a project's choices for the hooks
// (who commits and pushes, which branches may be deleted, usage thresholds, docs cap exemptions,
// formatters). The scripts hold mechanism only, so update-from-hatchery can patch them without
// touching a project's policy. Schema and fail modes: docs/design/decisions/0006-hook-policy-file.md.
//
// load() returns { policy, error, source }:
//   - file absent: policy = DEFAULTS (Hatchery's own behaviour), error = null.
//   - file present and valid: each key given replaces the default for that key.
//   - file unreadable, not JSON, an unknown key, or a value of the wrong type: policy = DEFAULTS,
//     error = a one-line description. Each hook decides what an error means (git-guard denies
//     every git command; the others fall back to DEFAULTS and say so).
// Keys starting with "$" (for example "$comment") are ignored at any level.
//
// Debug: node .claude/hooks/policy.js [file]  prints the effective policy, or the error and exit 1.
// Test seam: HOOK_POLICY_FILE names another file (test-hooks.js sets it; unset in normal use).
'use strict';
const fs = require('fs');
const path = require('path');

const DEFAULTS = {
  git: {
    commit: { subagents: ['runner'], protectedBranches: [] },
    push: { subagents: ['pusher'], branches: ['*'], mainSession: true, forceWithLease: [] },
    branchDelete: { branches: [], requireMergedPr: false },
    hooksPath: null,
  },
  usage: {
    hold: 50,
    deny: 80,
    staleSeconds: 7200,
    whenStale: 'allow',
    holdAdvice: 'Do not start new subagents. Let in-flight agents finish, run the handoff skill, commit from the main session, then report.',
    denyAdvice: 'Stop all work, run the handoff skill, commit WIP from the main session, end the turn.',
  },
  edit: { capExempt: [], formatters: {} },
};

const isStrings = (v) => Array.isArray(v) && v.every((s) => typeof s === 'string' && s !== '');
// Type checks for leaf values; each returns true when the value is acceptable.
const TYPES = {
  who: (v) => v === 'any' || v === 'none' || isStrings(v),
  patterns: isStrings,
  bool: (v) => typeof v === 'boolean',
  pathOrNull: (v) => v === null || (typeof v === 'string' && v !== ''),
  percent: (v) => typeof v === 'number' && v >= 0 && v <= 100,
  seconds: (v) => Number.isInteger(v) && v > 0,
  stale: (v) => v === 'allow' || v === 'deny',
  text: (v) => typeof v === 'string' && v !== '',
  formatters: (v) => v !== null && typeof v === 'object' && !Array.isArray(v) &&
    Object.keys(v).every((k) => /^\.[A-Za-z0-9]+$/.test(k) && isStrings(v[k]) && v[k].length > 0),
};
const SCHEMA = {
  git: {
    commit: { subagents: 'who', protectedBranches: 'patterns' },
    push: { subagents: 'who', branches: 'patterns', mainSession: 'bool', forceWithLease: 'patterns' },
    branchDelete: { branches: 'patterns', requireMergedPr: 'bool' },
    hooksPath: 'pathOrNull',
  },
  usage: { hold: 'percent', deny: 'percent', staleSeconds: 'seconds', whenStale: 'stale', holdAdvice: 'text', denyAdvice: 'text' },
  edit: { capExempt: 'patterns', formatters: 'formatters' },
};

// Walks value against schema, merging over defaults. Throws on the first problem.
function merge(schema, value, defaults, where) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) throw new Error(`${where || 'top level'} must be an object`);
  const out = {};
  for (const k of Object.keys(schema)) out[k] = defaults[k];
  for (const k of Object.keys(value)) {
    if (k.startsWith('$')) continue;
    const at = where ? `${where}.${k}` : k;
    if (!(k in schema)) throw new Error(`unknown key ${at}`);
    if (typeof schema[k] === 'string') {
      if (!TYPES[schema[k]](value[k])) throw new Error(`${at} has the wrong type or value (${JSON.stringify(value[k])})`);
      out[k] = value[k];
    } else {
      out[k] = merge(schema[k], value[k], defaults[k], at);
    }
  }
  return out;
}

function policyFile() {
  return process.env.HOOK_POLICY_FILE || path.join(__dirname, 'policy.json');
}

function load(file) {
  const source = file || policyFile();
  let text;
  try { text = fs.readFileSync(source, 'utf8'); } catch (e) {
    if (e && e.code === 'ENOENT') return { policy: DEFAULTS, error: null, source: null };
    return { policy: DEFAULTS, error: `cannot read ${source}: ${e.message}`, source };
  }
  try {
    const policy = merge(SCHEMA, JSON.parse(text.charCodeAt(0) === 0xfeff ? text.slice(1) : text), DEFAULTS, '');
    if (policy.usage.hold > policy.usage.deny) throw new Error('usage.hold is above usage.deny');
    return { policy, error: null, source };
  } catch (e) {
    return { policy: DEFAULTS, error: `${source}: ${e.message}`, source };
  }
}

// True when name matches one of the patterns; "*" matches any run of characters, "/" included.
function matches(patterns, name) {
  return patterns.some((p) => new RegExp(`^${p.split('*').map((s) => s.replace(/[.+?^${}()|[\]\\]/g, '\\$&')).join('.*')}$`).test(name));
}

module.exports = { load, matches, DEFAULTS };

if (require.main === module) {
  const r = load(process.argv[2]);
  if (r.error) { process.stderr.write(`policy: ${r.error}\n`); process.exit(1); }
  process.stdout.write(`source: ${r.source || '(none; defaults)'}\n${JSON.stringify(r.policy, null, 2)}\n`);
}
