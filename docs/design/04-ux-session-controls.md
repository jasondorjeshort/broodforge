# UX: the session's controls

Sibling of `04-ux.md`, under its Conventions, date and status: what a running session lets its driver change
from the venue (the policy mode, the model and effort, the idle timeout) and the header's menus that offer the
first three. The settled sections moved here by topic when 04 reached the size cap; each numbered
`Owner n:` marker is answered in 04's list at the end. "Mode" throughout is Broodforge's own policy mode
(`asks first` or `acts freely`, ADR 0020), never the engine's permission mode, which nothing here changes.

## Policy mode

The channel's default or the driver's switch. Prompts are answered on the host inside reach; the
trace shows every call, so policy mode reads like prompt mode without the waits. Switch: `/session
mode asks-first|acts-freely`, driver only, effective at the next turn (ADR 0020); the header and footer show the current mode.

```
From the next turn: acts freely. Broodforge answers permission prompts itself, inside this
channel's reach (code). Switch back with /session mode asks-first.
```

Owner 7: the names people see are `asks first` / `acts freely (within reach)`.

## Model and effort (vision Q 5)

A switch is a per-process flag; context survives it (`--resume`, ADR 0020). Owner 9 (channel configuration only) was
reversed 2026-10-22, after the first live run (R-29): the driver picks at start, `/session start
model:opus|sonnet|haiku|fable` (else the channel's `model`), and switches in the venue from the next turn, like mode.
`/model` alone replies `Model: Opus (this channel's default). Switch with /model opus, sonnet, haiku or fable.`; `/model
sonnet` (or `/session model sonnet`) and `/session effort low|medium|high|xhigh|max` post `From the next turn: Sonnet.` or
`From the next turn: effort high.`; else `Not a model: <value>. Models: opus, sonnet, haiku, fable.` The header gains
`Model: Opus (this channel's default). Effort: high.`, edited on a switch. A switch of the model, the effort or the mode
by anyone but the driver, typed or picked from the header's menus, is refused ephemerally in the owner's words
(2026-10-24): `Only the driver, <name>, can change the model.`, `... can change the effort.`, `... can change the
mode.`; the CLI keeps the service's `model needs the driver, <name>`. Rules: `05-debug-interface-discord.md`.

## Idle timeout

The channel's `idleTimeoutMinutes`, else the instance default; 0 or absent is never, the default since 2026-10-23 (the
owner: sessions should not end on their own; internally never is a year, the cap on any timeout, and a longer value is
capped at a year). The header then reads `Never ends for inactivity.` in place of `Ends after
60 minutes without activity.` The driver changes a running session's: `/session timeout never|<minutes>|<duration>`
(`30`, `90m`, `2h`; at least a minute), effective now, the clock restarting if the session is waiting. It posts `Ends
after 120 minutes without activity.` or `Never ends for inactivity.`, with ` Capped at a year.` appended when the value was
above a year, and edits the header; else `Not a timeout: <value>. Give never, a number of
minutes, or a duration such as 90m or 2h.`

## The header's menus

The owner, 2026-10-24: under the header's text sit three select menus (Discord's string selects), one action
row each, in the order model, effort, mode, and under them the `End session` row. A menu is a dropdown that
shows one option closed and the list open:

`[#s0001-jason] <the header's text>  [Sonnet ▾] [high ▾] [asks first ▾]  [End session]`

No menu has a label. Each marks the current value as its default option, so the closed menu reads as the
setting and the header's text keeps the reason (`this channel's default`, `from turn 3`). Only a menu with
nothing set shows a placeholder: the model's is its header line's own words, `Model: the engine's own`; the
effort menu has no header line to echo (the header gains `Effort:` only once a level is set) and reads
`Effort: the engine's own` to match; mode is always set. The options: the model menu lists policy's model map, one option per alias, each
labelled as the footer names the model (`Opus`, `Sonnet`); the effort menu lists the levels `policy.Efforts`
holds, labelled by the level, so a level added there appears here without a change to this doc; the mode menu
has `asks first` and `acts freely (within reach)` (Owner 7). A model the map does not name (a channel's `model`
key may hold one) marks no option, and that menu's placeholder is `Model: <id>`, the header's words again.

A pick is the slash command by another route: `/session model`, `/session effort` or `/session mode` with the
picked value, under the same right (the driver), the same refusal, the same audit row and the same post in the
venue (`From the next turn: Sonnet.`, `From the next turn: effort high.`, the mode switch's text above). Model
and effort take effect from the next turn, as a typed switch does. The header is then edited in place and the
menu's default follows the new value; every header edit carries all three menus from the current state, so a
switch of one never stales another. A pick of the value already set is a switch like any other (the post, the
audit row with `from` equal to `to`, the header edit), exactly what typing `/session model sonnet` on Sonnet
does; nothing is swallowed, and the header edit is what puts the picker's menu back in step. Whether Discord
sends an interaction at all for a pick of the option already marked is assumed, not observed; the owner's
step under `serve --discord live` settles it (`05-debug-interface-discord.md`, The header's menus).

A pick by someone without the right gets the slash command's ephemeral refusal (`Only the driver, <name>, can change
the model.`, Model and effort above) and posts nothing in the venue.
Discord is assumed to keep a pick on the picker's own screen until the message is edited, so the bot then edits
the header again, unchanged, and every screen shows the true value; the same owner's step observes it, and if
an unchanged edit does not reset the picker's screen the fallback is two edits a second apart, the first
without the menus and the second with them back, which re-creates the components. When the session ends, the
header is edited once more with every menu and the `End session` button removed; its text stays as the record.
A pick that still arrives (a client that had not refreshed, in the minutes before the venue is deleted) is
refused by the session itself, as the typed switches (`/session mode|model|effort`, `/model <alias>`), a venue
message, an answer, `Stop` and `/session invite` are once a session is ending or ended: the ephemeral `This
session has ended.`, no post, no header edit, nothing written to the session but the refusal's audit row. The
closed check comes before every other check (the driver's, a pending call's, a running turn's), so a pick by
someone who is not the driver gets this reply too, not the driver refusal. After a restart the bot holds every
interrupted session (ADR 0026) and its header keeps its menus and `End session` (owner-approved 2026-10-24): the button and `/session end`
hand the session off, and every other action, a pick included, is refused with 04's `interrupted refusal`
row, `This session was interrupted by a restart. /session end hands off its work.`
