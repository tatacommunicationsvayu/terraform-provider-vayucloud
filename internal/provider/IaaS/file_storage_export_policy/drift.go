// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_export_policy

import "strings"

// clientIPInVolumeClients reports whether clientIP appears in the platform client list.
func clientIPInVolumeClients(clients []string, clientIP string) bool {
	want := strings.TrimSpace(clientIP)
	if want == "" {
		return false
	}
	for _, c := range clients {
		if strings.TrimSpace(c) == want {
			return true
		}
	}
	return false
}

// driftLookupEnabled is true when both optional lookup fields are set for fetchVolumeDetails.
func driftLookupEnabled(engagementIDSet bool, fileServerID string) bool {
	return engagementIDSet && strings.TrimSpace(fileServerID) != ""
}
