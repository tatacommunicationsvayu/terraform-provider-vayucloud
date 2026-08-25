// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_export_policy

import "testing"

func TestExportPolicyID(t *testing.T) {
	t.Parallel()
	got := exportPolicyID("383464", "10.0.0.1")
	want := "383464/10.0.0.1"
	if got != want {
		t.Fatalf("exportPolicyID() = %q, want %q", got, want)
	}
}
