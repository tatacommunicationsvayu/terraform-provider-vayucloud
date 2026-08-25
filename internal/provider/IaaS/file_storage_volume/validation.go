// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

import (
	"context"
	"fmt"

	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/file_server"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/nas"
)

// ValidateNASOrder checks that a NAS base order exists for the given engagement, endpoint, and file storage type.
func ValidateNASOrder(c *client.Client, ctx context.Context, engagementID, endpointID int64, fileStorageType string) error {
	return nas.ValidateNASOrder(c, ctx, engagementID, endpointID, fileStorageType)
}

// ValidateNASOrderForFileServer resolves engagement, endpoint, and file storage type from the parent file server.
func ValidateNASOrderForFileServer(c *client.Client, ctx context.Context, fileServerID string) error {
	detail, err := file_server.ReadFileServerDetail(c, ctx, fileServerID)
	if err != nil {
		return err
	}
	if detail.EngagementID == 0 || detail.EndpointID == 0 {
		return fmt.Errorf("file server %q is missing engagement_id or endpoint_id required for NAS order validation", fileServerID)
	}
	fsType := detail.FileStorageType
	if fsType == "" {
		fsType = detail.FileServerType
	}
	if fsType == "" {
		return fmt.Errorf("file server %q is missing file_storage_type required for NAS order validation", fileServerID)
	}
	return nas.ValidateNASOrder(c, ctx, detail.EngagementID, detail.EndpointID, fsType)
}
