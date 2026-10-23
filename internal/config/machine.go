// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"os"
	"os/user"
	"strings"
)

// MachineLiteral is a literal the push scan refuses as a name wherever a session commit holds
// it, as the base's privacy scan does (ADR 0025 rule 7): the OS user name or the host name.
type MachineLiteral struct {
	What  string // OSUserName or HostName
	Value string
}

// What a machine literal is.
const (
	OSUserName = "the OS user name"
	HostName   = "the host name"
)

// MachineLiterals reads the machine's literals: the OS user name, bare (a Windows account's
// domain prefix dropped, as the base's os.userInfo gives it), then the host name; the push scan
// refuses both as names (git.Privacy.Machine). Nothing checks them at load: a service account
// whose name is the project's is lifted by the owner's privacy.allow, and a hit shows as the
// scan's privacy row, never printing the value. Tests replace it, so none reads the machine.
var MachineLiterals = readMachineLiterals

// machineLiteralsOff is set to "true" only by the test harness's build of the binary
// (clitest, through -ldflags -X), so a test's child process reads no machine literal either.
var machineLiteralsOff string

func readMachineLiterals() []MachineLiteral {
	if machineLiteralsOff == "true" {
		return nil
	}
	var out []MachineLiteral
	if u, err := user.Current(); err == nil {
		name := u.Username
		if i := strings.LastIndexByte(name, '\\'); i >= 0 {
			name = name[i+1:]
		}
		out = append(out, MachineLiteral{OSUserName, name})
	}
	if h, err := os.Hostname(); err == nil {
		out = append(out, MachineLiteral{HostName, h})
	}
	return out
}
