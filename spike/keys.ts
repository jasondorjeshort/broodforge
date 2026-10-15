// SPDX-License-Identifier: GPL-3.0-or-later
// C36: what `.claude.json` holds after a login (docs/design/07-spike-plan.md, C36).
//
//   node keys.js <file.json>
//
// Prints one line per leaf: its key path and its value, except that any value whose JSON text is
// longer than 40 characters prints as <long>. Arrays print their length and their elements' paths.

import { readFileSync } from 'node:fs';
import { die } from './common.js';

function walk(v: unknown, path: string, out: string[]): void {
  if (Array.isArray(v)) {
    out.push(`${path} [array of ${v.length}]`);
    v.forEach((x, i) => walk(x, `${path}[${i}]`, out));
  } else if (typeof v === 'object' && v !== null) {
    const keys = Object.keys(v);
    if (keys.length === 0) out.push(`${path} {}`);
    for (const k of keys) walk((v as Record<string, unknown>)[k], path ? `${path}.${k}` : k, out);
  } else {
    const text = JSON.stringify(v);
    out.push(`${path} = ${text.length > 40 ? '<long>' : text}`);
  }
}

const file = process.argv[2] ?? die('usage: node keys.js <file.json>');
let data: unknown;
try {
  data = JSON.parse(readFileSync(file, 'utf8'));
} catch (e) {
  die(`keys: ${(e as Error).message}`);
}
const out: string[] = [];
walk(data, '', out);
process.stdout.write(out.join('\n') + '\n');
