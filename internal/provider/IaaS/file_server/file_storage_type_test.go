// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import "testing"

func TestCanonicalFileServerStorageType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "NAS-NFS", want: "NAS-NFS"},
		{in: "NFS", want: "NAS-NFS"},
		{in: "NAS-CIFS", want: "NAS-CIFS"},
		{in: "CIFS", want: "NAS-CIFS"},
		{in: "EDED", want: ""},
		{in: "eded", want: ""},
		{in: "SHR", want: ""},
		{in: "", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := canonicalFileServerStorageType(tc.in); got != tc.want {
				t.Fatalf("canonicalFileServerStorageType(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeFileServerDetailIgnoresComponentType(t *testing.T) {
	t.Parallel()

	d := FileServerDetail{FileServerType: "eded"}
	normalizeFileServerDetail(&d)
	if d.FileStorageType != "" {
		t.Fatalf("FileStorageType = %q, want empty", d.FileStorageType)
	}

	d = FileServerDetail{FileStorageType: "NAS-NFS", FileServerType: "eded"}
	normalizeFileServerDetail(&d)
	if d.FileStorageType != "NAS-NFS" {
		t.Fatalf("FileStorageType = %q, want NAS-NFS", d.FileStorageType)
	}
}
