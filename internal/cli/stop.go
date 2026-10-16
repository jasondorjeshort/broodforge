// SPDX-License-Identifier: GPL-3.0-or-later

package cli

// stopCmd is `broodforge stop [--timeout 30]` (05-debug-interface.md, Bot lifecycle): 03's
// shutdown over the control socket. The bot interrupts every running turn, marks every open
// session interrupted and prints `stopped  <sessions interrupted>`, then closes; past the
// timeout it refuses stop-timeout naming the sessions still open, and exits 1 itself.
var stopCmd = Command{
	Name:  "stop",
	Class: ClassWrite,
	Flags: []Flag{
		{Name: "timeout", Kind: FlagInt, Default: "30", Value: "<seconds>", Usage: "how long the running turns may take to stop"},
	},
	Columns: []string{"result", "interrupted"},
}

func init() { commands = append(commands, &stopCmd) }
