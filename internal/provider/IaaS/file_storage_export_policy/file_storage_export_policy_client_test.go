// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_export_policy

import "testing"

func TestIsTransientNASAttachError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "policy missing", err: fmtError("No policyId found for vserverId: 1"), want: true},
		{name: "volume missing", err: fmtError("No matching NAS Volume found for given IDs"), want: true},
		{name: "permanent", err: fmtError("invalid client ip"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isTransientNASAttachError(tc.err); got != tc.want {
				t.Fatalf("isTransientNASAttachError() = %v, want %v", got, tc.want)
			}
		})
	}
}

type fmtError string

func (e fmtError) Error() string { return string(e) }
