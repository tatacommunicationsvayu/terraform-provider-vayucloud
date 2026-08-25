// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import "strings"

// portalComponentTypes are endpoint component categories — not vserver file storage types.
var portalComponentTypes = map[string]struct{}{
	"EDED": {},
	"SHR":  {},
}

// canonicalFileServerStorageType maps portal read payloads to the values used in Terraform
// (e.g. createNasVserver / vayucloud_file_server.file_storage_type). Returns "" when the
// portal value is a component type (EDED/SHR) or otherwise not a storage type.
func canonicalFileServerStorageType(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if _, isComponent := portalComponentTypes[strings.ToUpper(s)]; isComponent {
		return ""
	}
	switch strings.ToUpper(s) {
	case "NFS", "NAS-NFS":
		return "NAS-NFS"
	case "CIFS", "NAS-CIFS":
		return "NAS-CIFS"
	default:
		return s
	}
}
