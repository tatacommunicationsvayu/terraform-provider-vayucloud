// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

import (
	"regexp"
	"strconv"
	"strings"
)

var nasVolumeSizeGBPattern = regexp.MustCompile(`(\d+)`)

// parseNasVolumeSizeGB extracts whole GB from portal size fields (40, "40", "40GB", "40 GB").
func parseNasVolumeSizeGB(raw any) int64 {
	s := strings.TrimSpace(jsonScalarToString(raw))
	if s == "" {
		return 0
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 {
		return int64(f)
	}
	m := nasVolumeSizeGBPattern.FindStringSubmatch(strings.ToUpper(s))
	if len(m) < 2 {
		return 0
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || n < 1 {
		return 0
	}
	return n
}
