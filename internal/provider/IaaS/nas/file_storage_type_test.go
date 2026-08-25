// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package nas

import "testing"

func TestNormalizeFileStorageTypeForOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "NAS-NFS", want: "NFS"},
		{in: "nas-nfs", want: "NFS"},
		{in: "NAS-CIFS", want: "CIFS"},
		{in: "NFS", want: "NFS"},
		{in: "PNFS", want: "PNFS"},
		{in: "PFS", want: "PFS"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeFileStorageTypeForOrder(tc.in); got != tc.want {
				t.Fatalf("NormalizeFileStorageTypeForOrder(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
