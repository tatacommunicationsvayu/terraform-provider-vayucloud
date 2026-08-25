// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

import "testing"

func TestParseNasVolumeSizeGB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   any
		want int64
	}{
		{in: 40, want: 40},
		{in: "40", want: 40},
		{in: "40GB", want: 40},
		{in: "40 GB", want: 40},
		{in: float64(40), want: 40},
		{in: "", want: 0},
		{in: "invalid", want: 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(jsonScalarToString(tc.in), func(t *testing.T) {
			t.Parallel()
			if got := parseNasVolumeSizeGB(tc.in); got != tc.want {
				t.Fatalf("parseNasVolumeSizeGB(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestPopulateVolumeLookupFromJSONSizeGb(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"volCi":405932,"volumeName":"nfsVolume1","sizeGb":"40GB"}`)
	out := &FileStorageVolumeLookupDetail{}
	if err := populateVolumeLookupFromJSON(raw, out); err != nil {
		t.Fatal(err)
	}
	if out.SizeGb != 40 {
		t.Fatalf("SizeGb = %d, want 40", out.SizeGb)
	}
}
