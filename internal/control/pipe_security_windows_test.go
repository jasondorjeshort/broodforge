// SPDX-License-Identifier: GPL-3.0-or-later

package control

import (
	"testing"

	"golang.org/x/sys/windows"
)

// pipeSDDL, given a user SID that is not this process's, parses to a protected DACL of exactly
// SYSTEM, Administrators and that SID. It sees no token; the descriptor a served pipe actually
// gets is TestPipeDACLIsSystemAdministratorsAndUser's.
func TestPipeSDDL(t *testing.T) {
	const user = "S-1-5-21-1000000001-1000000002-1000000003-1001"
	sddl := pipeSDDL(user)
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatalf("%s: %v", sddl, err)
	}
	want := "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;" + user + ")"
	if got := sd.String(); got != want {
		t.Errorf("descriptor = %s, want %s", got, want)
	}
	if ctl, _, err := sd.Control(); err != nil || ctl&windows.SE_DACL_PROTECTED == 0 {
		t.Errorf("DACL not protected: control %#x, %v", ctl, err)
	}
}
