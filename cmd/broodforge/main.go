// SPDX-License-Identifier: GPL-3.0-or-later

// Command broodforge is the one binary: the bot, the hook and every debug command.
package main

import (
	"os"

	"github.com/jasondorjeshort/broodforge/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
