// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package nas

import "strings"

// NormalizeFileStorageTypeForOrder maps portal file storage types to the value used for NAS base-order matching
// (see NasServiceImpl.normalizeFileStorageTypeForOrder / resolveFileStorageTypeSuffix).
func NormalizeFileStorageTypeForOrder(fileStorageType string) string {
	switch strings.ToUpper(strings.TrimSpace(fileStorageType)) {
	case "NAS-NFS":
		return "NFS"
	case "NAS-CIFS":
		return "CIFS"
	case "PFS", "PNFS", "DDN-NAS":
		return strings.ToUpper(strings.TrimSpace(fileStorageType))
	default:
		return strings.TrimSpace(fileStorageType)
	}
}
