// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestParseNasVolumeSizePrecheckResponse(t *testing.T) {
	t.Parallel()

	current20 := int64(20)

	tests := []struct {
		name      string
		body      string
		wantAllow bool
		wantCurr  *int64
		wantErr   bool
	}{
		{
			name: "grow allowed",
			body: `{"status":"success","data":{"isAllowed":true,"currentSize":20,"requestedSize":50}}`,
			wantAllow: true, wantCurr: &current20,
		},
		{
			name: "shrink rejected",
			body: `{"status":"success","data":{"isAllowed":false,"currentSize":50,"requestedSize":20}}`,
			wantAllow: false, wantCurr: int64Ptr(50),
		},
		{
			name: "equal size rejected",
			body: `{"status":"success","data":{"isAllowed":false,"currentSize":50,"requestedSize":50}}`,
			wantAllow: false, wantCurr: int64Ptr(50),
		},
		{
			name: "null current size",
			body: `{"status":"success","data":{"isAllowed":false,"currentSize":null,"requestedSize":50}}`,
			wantAllow: false, wantCurr: nil,
		},
		{
			name:    "portal failure",
			body:    `{"status":"failed","message":"Failed","data":{"isAllowed":false,"error":"Failed to check NAS volume size"}}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseNasVolumeSizePrecheckResponse(123, []byte(tc.body))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.IsAllowed != tc.wantAllow {
				t.Fatalf("IsAllowed = %v, want %v", got.IsAllowed, tc.wantAllow)
			}
			if !int64PtrEqual(got.CurrentSize, tc.wantCurr) {
				t.Fatalf("CurrentSize = %v, want %v", ptrStr(got.CurrentSize), ptrStr(tc.wantCurr))
			}
		})
	}
}

func TestNasVolumeSizePrecheckDisallowError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		check   NasVolumeSizePrecheck
		contain string
	}{
		{
			name:    "shrink",
			check:   NasVolumeSizePrecheck{VolumeID: 1, RequestedSize: 20, CurrentSize: int64Ptr(50)},
			contain: "must be greater than current portal size 50 GB",
		},
		{
			name:    "unknown current",
			check:   NasVolumeSizePrecheck{VolumeID: 1, RequestedSize: 50, CurrentSize: nil},
			contain: "could not determine current size",
		},
		{
			name:    "portal error",
			check:   NasVolumeSizePrecheck{VolumeID: 1, PortalError: "NAS volume not found for volume id 1"},
			contain: "NAS volume not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.check.disallowError()
			if !errors.Is(err, ErrNasVolumeResizeNotAllowed) {
				t.Fatalf("expected ErrNasVolumeResizeNotAllowed, got %v", err)
			}
			if !strings.Contains(err.Error(), tc.contain) {
				t.Fatalf("error %q should contain %q", err.Error(), tc.contain)
			}
		})
	}
}

func int64Ptr(v int64) *int64 { return &v }

func int64PtrEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func ptrStr(p *int64) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *p)
}
