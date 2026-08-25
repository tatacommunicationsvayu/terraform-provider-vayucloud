// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

import "testing"

func TestAdoptNasVolumeOutOfBandSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured int64
		state      int64
		live       int64
		wantSize   int64
		wantAdopt  bool
	}{
		{name: "no out-of-band change", configured: 20, state: 20, live: 20, wantSize: 20, wantAdopt: false},
		{name: "portal grew state refreshed", configured: 20, state: 50, live: 50, wantSize: 50, wantAdopt: true},
		{name: "stale state live larger", configured: 20, state: 20, live: 50, wantSize: 50, wantAdopt: true},
		{name: "intentional resize up", configured: 80, state: 50, live: 50, wantSize: 80, wantAdopt: false},
		{name: "shrink attempt blocked", configured: 10, state: 50, live: 50, wantSize: 50, wantAdopt: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotSize, gotAdopt := adoptNasVolumeOutOfBandSize(tc.configured, tc.state, tc.live)
			if gotSize != tc.wantSize || gotAdopt != tc.wantAdopt {
				t.Fatalf("adoptNasVolumeOutOfBandSize(%d,%d,%d) = (%d,%v), want (%d,%v)",
					tc.configured, tc.state, tc.live, gotSize, gotAdopt, tc.wantSize, tc.wantAdopt)
			}
		})
	}
}
