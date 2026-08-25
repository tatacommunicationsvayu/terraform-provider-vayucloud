// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_export_policy

import "testing"

func TestClientIPInVolumeClients(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		clients  []string
		clientIP string
		want     bool
	}{
		{name: "exact match", clients: []string{"10.0.0.1"}, clientIP: "10.0.0.1", want: true},
		{name: "trimmed match", clients: []string{" 10.0.0.2 "}, clientIP: "10.0.0.2", want: true},
		{name: "missing", clients: []string{"10.0.0.1"}, clientIP: "10.0.0.2", want: false},
		{name: "empty clients", clients: nil, clientIP: "10.0.0.1", want: false},
		{name: "empty client ip", clients: []string{"10.0.0.1"}, clientIP: "  ", want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := clientIPInVolumeClients(tc.clients, tc.clientIP); got != tc.want {
				t.Fatalf("clientIPInVolumeClients() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDriftLookupEnabled(t *testing.T) {
	t.Parallel()

	if !driftLookupEnabled(true, "123") {
		t.Fatal("expected enabled when engagement and file server are set")
	}
	if driftLookupEnabled(false, "123") {
		t.Fatal("expected disabled without engagement")
	}
	if driftLookupEnabled(true, "") {
		t.Fatal("expected disabled without file server")
	}
	if driftLookupEnabled(true, "  ") {
		t.Fatal("expected disabled for blank file server")
	}
}
