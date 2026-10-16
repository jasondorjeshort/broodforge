// SPDX-License-Identifier: GPL-3.0-or-later

package cli

// controlEcho is the hidden probe of the control socket (05-cli-conventions.md, Probe and
// harness):
// `broodforge control echo [--text <s>] [--events <n>] [--fail <code>] [--expect <value>]`.
// It streams <n> events {"seq":i,"text":<s>} (text: `<s>  <i>`), then prints the result
// {"text":<s>} (text: `<s>`); --fail <code> makes the bot refuse with that code after the
// events. Every server answers it (control.CmdEcho), so it proves a bot is listening and
// exercises the wire format, streaming, the error line and --expect.
var controlEcho = Command{
	Name:   "control echo",
	Class:  ClassWrite,
	Hidden: true,
	Expect: true,
	Flags: []Flag{
		{Name: "text", Kind: FlagString, Default: "ok", Value: "<s>", Usage: "the text echoed back"},
		{Name: "events", Kind: FlagInt, Default: "0", Value: "<n>", Usage: "events streamed before the result"},
		{Name: "fail", Kind: FlagString, Value: "<code>", Usage: "refuse with this code after the events"},
	},
	Columns:      []string{"text"},
	EventColumns: []string{"text", "seq"},
}
