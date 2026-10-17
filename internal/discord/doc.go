// SPDX-License-Identifier: GPL-3.0-or-later

// Package discord is 03's discord component: rendering session events into posts and edits
// (View), the outbound mask, the coalescer that paces them (one create or edit per second per
// session, 50 requests per second per bot), the outbox a dry run writes them to, custom ids,
// slash-command parsing, the key-paste scan and the inbound handlers `discord inject` feeds.
// The gateway is P12's. disgo's types appear in this package and nowhere else (ADR 0023).
package discord
