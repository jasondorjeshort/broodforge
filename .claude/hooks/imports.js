// SPDX-License-Identifier: GPL-3.0-or-later
// The one parser of Claude Code's @ imports in CLAUDE.md and the files it loads (ADR 0016, item 4;
// spec: docs/design/decisions/0016-update-procedure.md, Checks). policy.js --claude-check,
// post-edit.js and ownership.js --report call it, so the edit, the push and the update agree.
//
// The rule, following the loader (https://code.claude.com/docs/en/memory#import-additional-files):
//   - an @ counts at line start or after whitespace, so "See @README" imports README;
//   - never inside a fenced block (``` or ~~~, any indent) or a code span (`...`, ``...``);
//   - the token runs to the next whitespace, with trailing . , ; : ) stripped;
//   - it resolves from the importing file's directory (~/ from the home directory, an absolute
//     path as written);
//   - imports nest four hops: the start file is hop 0, its imports hop 1; a file at hop 4 is loaded
//     but its own imports are not.
// The sentinel procedure in the agreement-loading reference entry checks this against the loader;
// a mismatch is a fix here.
//
// parse(text) -> [{ line, token }]  line 1-based, token without the @.
// resolve(root, file, opts) -> [{ file, line, token, target, exists, hop }] for every import reached
//   from file (repo-relative, forward slashes) within opts.hops (default 4), each file read once,
//   breadth first. target is repo-relative with forward slashes, or absolute when outside root.
//   opts.skip: targets listed but not followed (the handoff chain for the 190 count).
// Node built-ins only (ADR 0013). An unreadable or absent file yields no imports, never a throw.
'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');

const HOPS = 4;
const SPAN = '\u0000'; // stands in for code-span text: not whitespace, so an @ after a span does not count

// Replaces each code span (a run of n backticks up to the next run of exactly n) with SPAN
// characters; an unmatched run stays literal, as CommonMark has it.
function blankSpans(line) {
  let out = '';
  let i = 0;
  while (i < line.length) {
    if (line[i] !== '`') { out += line[i++]; continue; }
    let n = 0;
    while (line[i + n] === '`') n++;
    let j = i + n;
    let close = -1;
    while (j < line.length) {
      if (line[j] !== '`') { j++; continue; }
      let m = 0;
      while (line[j + m] === '`') m++;
      if (m === n) { close = j; break; }
      j += m;
    }
    if (close < 0) { out += line.slice(i, i + n); i += n; continue; }
    out += SPAN.repeat(close + n - i);
    i = close + n;
  }
  return out;
}

function parse(text) {
  const found = [];
  let fence = null; // { ch, len } while inside a fenced block
  const rows = String(text).replace(/\r/g, '').split('\n');
  rows.forEach((raw, k) => {
    const f = raw.match(/^[ \t]*(`{3,}|~{3,})(.*)$/);
    if (fence === null && f) { fence = { ch: f[1][0], len: f[1].length }; return; }
    if (fence !== null) {
      if (f && f[1][0] === fence.ch && f[1].length >= fence.len && f[2].trim() === '') fence = null;
      return;
    }
    const line = blankSpans(raw);
    for (const m of line.matchAll(/(^|\s)@(\S+)/g)) {
      let token = m[2];
      const cut = token.indexOf(SPAN);
      if (cut >= 0) token = token.slice(0, cut);
      token = token.replace(/[.,;:)]+$/, '');
      if (token !== '') found.push({ line: k + 1, token });
    }
  });
  return found;
}

const posix = (p) => p.split(path.sep).join('/');

// The target of token imported from file (repo-relative), as repo-relative or absolute.
function targetOf(root, file, token) {
  let abs;
  if (token.startsWith('~/')) abs = path.join(os.homedir(), ...token.slice(2).split('/'));
  else if (path.isAbsolute(token) || /^[A-Za-z]:[\\/]/.test(token)) abs = path.resolve(token);
  else {
    const rel = path.posix.normalize(path.posix.join(path.posix.dirname(file), token));
    if (rel !== '..' && !rel.startsWith('../')) return rel.replace(/^\.\//, '');
    abs = path.resolve(root, ...rel.split('/'));
  }
  const rel = path.relative(root, abs);
  return rel && !rel.startsWith('..') && !path.isAbsolute(rel) ? posix(rel) : posix(abs);
}

const absOf = (root, target) => (path.isAbsolute(target) || /^[A-Za-z]:\//.test(target) ? path.resolve(target) : path.join(root, ...target.split('/')));

function readOrNull(file) {
  try { return fs.readFileSync(file, 'utf8'); } catch (_) { return null; }
}

function resolve(root, file, opts) {
  const o = opts || {};
  const hops = o.hops === undefined ? HOPS : o.hops;
  const skip = new Set(o.skip || []);
  const out = [];
  const seen = new Set([file]);
  let level = [file];
  for (let hop = 1; hop <= hops && level.length > 0; hop++) {
    const next = [];
    for (const f of level) {
      const text = readOrNull(absOf(root, f));
      if (text === null) continue;
      for (const { line, token } of parse(text)) {
        const target = targetOf(root, f, token);
        let isFile = false;
        let exists = false;
        try { const s = fs.statSync(absOf(root, target)); exists = true; isFile = s.isFile(); } catch (_) { /* absent */ }
        out.push({ file: f, line, token, target, exists, hop });
        if (isFile && !skip.has(target) && !seen.has(target)) { seen.add(target); next.push(target); }
      }
    }
    level = next;
  }
  return out;
}

module.exports = { parse, resolve, absOf, HOPS };
