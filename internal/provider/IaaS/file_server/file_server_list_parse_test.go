// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestParseNASVserverListRowsEndpointBuckets(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{
		"EP_V2_MUM_BKC": [{"vserverId": 56444, "name": "MUM-BKCEXAVSNFS01"}],
		"EP_V2_BL": [{"vserverId": 57119, "name": "blEXAVSNFS03", "vserverDisplayName": "nfsFileServer1"}]
	}`)

	rows, err := parseNASVserverListRows(raw)
	if err != nil {
		t.Fatalf("parseNASVserverListRows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2 (all endpoint buckets merged)", len(rows))
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		entry, err := row.toEntry()
		if err != nil {
			t.Fatalf("toEntry: %v", err)
		}
		ids = append(ids, entry.VserverID)
	}
	slices.Sort(ids)
	want := []int64{56444, 57119}
	if !slices.Equal(ids, want) {
		t.Fatalf("vserver ids = %v, want %v", ids, want)
	}
}

func TestParseNASVserverListRowsFlatArray(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`[{"vserverId": 1, "name": "a"}, {"vserverId": 2, "name": "b"}]`)
	rows, err := parseNASVserverListRows(raw)
	if err != nil {
		t.Fatalf("parseNASVserverListRows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
}
