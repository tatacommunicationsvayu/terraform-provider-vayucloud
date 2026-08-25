// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

// adoptNasVolumeOutOfBandSize aligns configured size with portal reality when the NAS volume
// was grown outside Terraform (configuration lags behind state/API).
//
// Returns the size to use in plan and whether the plan value was adjusted.
// Shrinks (configured < actual) are not applied — actual size wins, matching
// virtualmachine_blockstorage behaviour (in-place decrease is not supported).
func adoptNasVolumeOutOfBandSize(configuredSize, stateSize, liveSize int64) (int64, bool) {
	actual := stateSize
	if liveSize > actual {
		actual = liveSize
	}
	if configuredSize < actual {
		return actual, true
	}
	return configuredSize, false
}
