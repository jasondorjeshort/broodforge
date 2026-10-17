#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// PostToolUse hook for the Write and Edit tools. Non-blocking: it never exits non-zero
// and never denies. It returns hookSpecificOutput.additionalContext when an edit to CLAUDE.md,
// .claude/agreement/*.md or .claude/project/*.md leaves a --claude-check FAIL (claudeCheck in
// policy.js: sections, the 40 and 190 caps, unresolved imports), or an edited docs/**/*.md file
// breaks a size cap, lacks required frontmatter, is missing from its directory index, or names a
// repository path that does not exist, and it runs the formatter for the file's extension if one
// is set. Every @ import is read by imports.js, the one parser.
// Spec: docs/design/decisions/0002-hook-specs.md section 2; policy: ADR 0006; CLAUDE.md: ADR 0016.
// Test harness: node .claude/hooks/test-hooks.js
'use strict';
const fs = require('fs');
const path = require('path');
const childProcess = require('child_process');
const { load, matches, docCap, lineCount, claudeCheck, DOCS_CAP, HANDOFF_CAP } = require('./policy');
const { resolve } = require('./imports');

// Project choices from .claude/hooks/policy.json (edit block); a malformed policy means the base
// defaults here, since this hook only advises.
const { edit } = load().policy;
// Formatters, keyed by file extension (lower case, with the dot). The value is an argv array;
// the edited file's absolute path is appended at run time. Empty in the base; a project sets
// edit.formatters, for example ".rs": ["rustfmt", "--edition", "2021"]. A non-zero exit or a
// 20 s timeout is reported back to the agent, never blocks.
const FORMATTERS = Object.assign({}, edit.formatters);
// Test seam: a JSON object in this variable is merged over FORMATTERS, so the harness
// can exercise the formatter path without a policy file. Unset in normal use.
if (process.env.POST_EDIT_FORMATTERS_JSON) {
  try { Object.assign(FORMATTERS, JSON.parse(process.env.POST_EDIT_FORMATTERS_JSON)); } catch (_) { /* ignore */ }
}

// The per-file caps (DOCS_CAP 300, HANDOFF_CAP 40) and which applies are docCap in policy.js, the
// one rule this hook and pusher check 7 share.
const HANDOFF_TOTAL_CAP = 120; // docs/HANDOFF.md plus every docs/handoff/*.md
// The CLAUDE.md caps (40, and 190 loaded) are CLAUDE_MD_CAP and CLAUDE_LOADED_CAP in policy.js.
const INDEX_EXEMPT = new Set(['README.md', 'HANDOFF.md']);

// True when the file starts with ---, closes with --- within the first 20 lines,
// and every key appears at line start between the two markers.
function hasFrontmatter(text, keys) {
  const lines = text.split(/\r?\n/);
  if (lines[0] !== '---') return false;
  let end = -1;
  for (let i = 1; i < Math.min(lines.length, 20); i++) {
    if (lines[i] === '---') { end = i; break; }
  }
  if (end < 0) return false;
  const body = lines.slice(1, end);
  return keys.every((k) => body.some((l) => l.startsWith(`${k}:`)));
}

function readOr(file, fallback) {
  try { return fs.readFileSync(file, 'utf8'); } catch (_) { return fallback; }
}

function checkDocs(root, rel, abs) {
  const msgs = [];
  const text = fs.readFileSync(abs, 'utf8');
  const n = lineCount(text);
  const parts = rel.split('/');
  const base = parts[parts.length - 1];

  const cap = docCap(edit, rel);
  if (cap === DOCS_CAP && n > cap) {
    msgs.push(`post-edit: ${rel} is ${n} lines; cap is ${cap} (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.`);
  }
  if (cap === HANDOFF_CAP && n > cap) {
    msgs.push(`post-edit: ${rel} is ${n} lines; cap is ${cap}. Move detail to docs/ideas/ or a finding.`);
  }
  const isStream = parts.length === 3 && parts[1] === 'handoff';
  const isHandoff = rel === 'docs/HANDOFF.md' || isStream;
  if (isHandoff) {
    let total = n;
    const others = [];
    if (rel !== 'docs/HANDOFF.md') others.push(path.join(root, 'docs', 'HANDOFF.md'));
    try {
      for (const f of fs.readdirSync(path.join(root, 'docs', 'handoff'))) {
        if (f.endsWith('.md') && !(isStream && f === base)) others.push(path.join(root, 'docs', 'handoff', f));
      }
    } catch (_) { /* no split directory */ }
    for (const f of others) total += lineCount(readOr(f, ''));
    if (total > HANDOFF_TOTAL_CAP) {
      msgs.push(`post-edit: the handoff loads ${total} lines in total (docs/HANDOFF.md plus docs/handoff/*.md); cap is ${HANDOFF_TOTAL_CAP}. Close a stream or trim.`);
    }
  }
  if (parts.length === 3 && parts[1] === 'findings' && base !== 'README.md' &&
      !hasFrontmatter(text, ['date', 'status', 'area'])) {
    msgs.push(`post-edit: ${rel} lacks finding frontmatter (date, status, area). See the finding skill.`);
  }
  if (parts.length === 3 && parts[1] === 'reference' && base !== 'README.md' &&
      !hasFrontmatter(text, ['verified'])) {
    msgs.push(`post-edit: ${rel} lacks verified: frontmatter. See .claude/rules/docs.md.`);
  }
  // Index check: a file under docs/<dir>/... must be named in a README.md, looking in its
  // own directory first and then in each parent up to docs/<dir>/ (ADRs, for example,
  // are listed in docs/design/README.md).
  if (isStream) {
    // docs/handoff/ has no README: a stream is listed when the map imports it.
    const map = resolve(root, 'docs/HANDOFF.md', { hops: 1 }).map((i) => i.target);
    if (!map.includes(rel)) {
      msgs.push(`post-edit: ${rel} is not imported by docs/HANDOFF.md. Add the line @handoff/${base} or delete the file.`);
    }
  } else if (parts.length >= 3 && !INDEX_EXEMPT.has(base)) {
    let listed = false;
    for (let depth = parts.length - 1; depth >= 2 && !listed; depth--) {
      const readme = path.join(root, ...parts.slice(0, depth), 'README.md');
      listed = readOr(readme, '').includes(base);
    }
    if (!listed) {
      const dir = parts.slice(1, -1).join('/');
      msgs.push(`post-edit: ${rel} is not listed in ${dir}/README.md. Add one line.`);
    }
  }
  // Closed records (proposals, decisions, findings by default) name paths as they were.
  const missing = matches(edit.anchorExempt, rel) ? [] : missingPaths(root, text);
  if (missing.length > 0) {
    const list = missing.map((m) => `\`${m}\``).join(', ');
    msgs.push(`post-edit: ${rel} names paths that do not exist: ${list}. Fix them, or mark the paragraph [no-check: <reason>] (.claude/rules/docs.md).`);
  }
  return msgs;
}

// Path anchors: a backtick-quoted token that looks like a repository path (it contains "/" and
// starts with an existing top-level directory such as docs or .claude) must exist, unless its
// paragraph, or its list item, carries [no-check: ...]. Fenced blocks (indented ones too) and
// placeholders (NNNN, <slug>, *) are skipped.
function missingPaths(root, text) {
  let tops;
  try {
    tops = new Set(fs.readdirSync(root, { withFileTypes: true })
      .filter((e) => e.isDirectory() && e.name !== '.git').map((e) => e.name));
  } catch (_) { return []; }
  const prose = text.replace(/\r/g, '').replace(/^[ \t]*(```|~~~)[^\n]*\n[\s\S]*?^[ \t]*\1[^\n]*$/gm, '');
  const units = prose.split(/\n[ \t]*\n/).flatMap((p) => p.split(/\n(?=[ \t]*(?:[-*+]|\d+[.)])[ \t])/));
  const missing = [];
  for (const para of units) {
    if (para.includes('[no-check:')) continue;
    for (const m of para.matchAll(/`([^`\s]+)`/g)) {
      const p = m[1].replace(/[#:].*$/, '');
      if (!p.includes('/') || /[*<>{}$]|NNNN|YYYY/.test(p) || !tops.has(p.split('/')[0])) continue;
      if (!missing.includes(p) && !fs.existsSync(path.join(root, ...p.split('/').filter(Boolean)))) missing.push(p);
    }
  }
  return missing;
}

// Import check for a handoff file: each of its own @ imports (imports.js: line start or after
// whitespace, outside code spans and fences) resolves from the importing file's directory (Claude
// Code's rule); an absent target loads nothing and says nothing.
function missingImports(root, rel) {
  return resolve(root, rel, { hops: 1 }).filter((i) => !i.exists).map((i) =>
    `post-edit: ${rel} imports @${i.token}, which resolves to ${i.target} and does not exist (an import resolves from the importing file's directory).`);
}

function runFormatter(argv, abs, rel) {
  const r = childProcess.spawnSync(argv[0], [...argv.slice(1), abs], { timeout: 20000, encoding: 'utf8' });
  if (!r.error && r.status === 0) return [];
  let detail = (r.stderr || '').split(/\r?\n/).filter((l) => l !== '').slice(0, 5).join('\n');
  if (!detail && r.error) detail = r.error.message;
  if (!detail && r.signal) detail = `killed by ${r.signal} (timeout 20 s?)`;
  return [`post-edit: formatter ${argv[0]} failed on ${rel}: ${detail}`];
}

function main(raw) {
  const input = JSON.parse(raw);
  const filePath = input && input.tool_input && input.tool_input.file_path;
  if (typeof filePath !== 'string' || filePath === '') return;
  const root = typeof input.cwd === 'string' && input.cwd ? input.cwd : process.cwd();
  const abs = path.resolve(root, filePath);
  const relRaw = path.relative(root, abs);
  const inside = relRaw && !relRaw.startsWith('..') && !path.isAbsolute(relRaw);
  const rel = inside ? relRaw.split(path.sep).join('/') : abs;

  const msgs = [];
  if (inside && rel.startsWith('docs/') && rel.endsWith('.md')) msgs.push(...checkDocs(root, rel, abs));
  if (inside && (rel === 'docs/HANDOFF.md' || /^docs\/handoff\/[^/]+\.md$/.test(rel))) {
    msgs.push(...missingImports(root, rel));
  }
  // The project CLAUDE.md and the files it is built from: the same FAIL lines as pusher check 7's
  // policy.js --claude-check, for the whole tree, since an edit to one file can break another's line.
  if (inside && (rel === 'CLAUDE.md' || /^\.claude\/(agreement|project)\/[^/]+\.md$/.test(rel))) {
    msgs.push(...claudeCheck(root, edit).lines);
  }
  const fmt = FORMATTERS[path.extname(abs).toLowerCase()];
  if (Array.isArray(fmt) && fmt.length > 0) msgs.push(...runFormatter(fmt, abs, rel));

  if (msgs.length > 0) {
    process.stdout.write(JSON.stringify({
      hookSpecificOutput: { hookEventName: 'PostToolUse', additionalContext: msgs.join('\n') },
    }));
  }
}

let raw = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => (raw += c));
process.stdin.on('end', () => {
  try { main(raw); } catch (_) { /* non-blocking by design: swallow everything */ }
  process.exit(0);
});
