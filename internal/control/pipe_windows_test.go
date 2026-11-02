// SPDX-License-Identifier: GPL-3.0-or-later

package control_test

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
)

// A pipe that exists but refuses the caller is connect-failed, not not-running: here the
// \\localhost form of a running bot's pipe, which PIPE_REJECT_REMOTE_CLIENTS refuses.
func TestRefusedPipeIsConnectFailed(t *testing.T) {
	h := controltest.Start(t)
	remote := strings.Replace(h.Addr, `\\.\`, `\\localhost\`, 1)
	_, err := control.Do(timeout(t), remote, control.CmdEcho, nil, nil)
	ce := wantCode(t, err, control.CodeConnectFailed)
	if !strings.HasPrefix(ce.Message, "cannot connect to "+remote+": ") {
		t.Errorf("message = %q", ce.Message)
	}
	_, err = control.Dial(timeout(t), remote)
	wantCode(t, err, control.CodeConnectFailed)
}

// A served pipe's DACL is exactly SYSTEM, Administrators and this process's user, protected:
// not the CreateNamedPipe default, which grants the token's default owner (Administrators under
// an S4U task) instead of the user and gives Everyone and Anonymous read. Run elevated, the test
// reproduces the S4U shape: the elevated token's default owner is Administrators, so the default
// would have left the user out.
func TestPipeDACLIsSystemAdministratorsAndUser(t *testing.T) {
	h := controltest.Start(t)
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	name16, err := windows.UTF16PtrFromString(h.Addr)
	if err != nil {
		t.Fatal(err)
	}
	// READ_CONTROL alone: the user's entry grants it. The open takes one pipe instance, which
	// the server sees as a client that sends nothing.
	ph, err := windows.CreateFile(name16, windows.READ_CONTROL, 0, nil, windows.OPEN_EXISTING,
		windows.SECURITY_SQOS_PRESENT|windows.SECURITY_IDENTIFICATION, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = windows.CloseHandle(ph) }()
	sd, err := windows.GetSecurityInfo(ph, windows.SE_KERNEL_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	// Round-trip the expectation so an alias SID (SY, LS, NS, LA as the user) renders as the
	// pipe's own does.
	wantSD, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sd.String(), wantSD.String(); got != want {
		t.Errorf("pipe DACL = %s, want %s", got, want)
	}
	// The pipe still serves after that open, on the next instance.
	if _, err := control.Do(timeout(t), h.Addr, control.CmdEcho, nil, nil); err != nil {
		t.Fatal(err)
	}
}
