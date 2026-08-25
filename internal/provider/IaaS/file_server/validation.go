// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_engagement"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_location"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/nas"
)

// ValidateEngagementAndEndpoint ensures the engagement exists for the account and the endpoint
// is linked to that engagement (same checks as vayucloud_network_firewall).
func ValidateEngagementAndEndpoint(c *client.Client, ctx context.Context, engagementID, endpointID int64) error {
	if err := account_engagement.ValidateEngagementExists(c, ctx, engagementID); err != nil {
		return err
	}
	return account_location.ValidateEndpointExists(c, ctx, engagementID, endpointID)
}

// ValidateFileServerExists checks that the vserver id is numeric and exists in the platform.
// Prefer ValidateFileServerInEngagement when engagement_id is known — fetchVserverDetails can fail
// for newly created vservers before engagement-endpoint-component links are visible.
func ValidateFileServerExists(c *client.Client, ctx context.Context, fileServerID string) error {
	fileServerID = strings.TrimSpace(fileServerID)
	if fileServerID == "" {
		return fmt.Errorf("file_server_id is required")
	}
	if _, err := strconv.ParseInt(fileServerID, 10, 64); err != nil {
		return fmt.Errorf("file_server_id must be numeric: %q", fileServerID)
	}
	_, err := GetFileServer(c, ctx, fileServerID)
	if err != nil {
		if errors.Is(err, ErrFileServerNotFound) {
			return fmt.Errorf("file server id %s not found", fileServerID)
		}
		return fmt.Errorf("failed to validate file server id %s: %w", fileServerID, err)
	}
	return nil
}

// ValidateFileServerInEngagement checks the vserver appears in getNASVservers for the engagement and endpoint.
func ValidateFileServerInEngagement(c *client.Client, ctx context.Context, engagementID, endpointID int64, fileServerID string) error {
	fileServerID = strings.TrimSpace(fileServerID)
	if fileServerID == "" {
		return fmt.Errorf("file_server_id is required")
	}
	if _, err := strconv.ParseInt(fileServerID, 10, 64); err != nil {
		return fmt.Errorf("file_server_id must be numeric: %q", fileServerID)
	}
	list, err := ListNASVservers(c, ctx, engagementID, endpointID)
	if err != nil {
		return fmt.Errorf("list NAS vservers for engagement %d endpoint %d: %w", engagementID, endpointID, err)
	}
	for _, item := range list.Items {
		if strconv.FormatInt(item.VserverID, 10) == fileServerID {
			return nil
		}
	}
	return fmt.Errorf("file server id %s not found for engagement %d endpoint %d (getNASVservers)", fileServerID, engagementID, endpointID)
}

// ValidateFileServerID ensures the platform file server exists and its engagement/endpoint are valid.
func ValidateFileServerID(c *client.Client, ctx context.Context, fileServerID string) error {
	fileServerID = strings.TrimSpace(fileServerID)
	if fileServerID == "" {
		return fmt.Errorf("file_server_id is required")
	}
	if _, err := strconv.ParseInt(fileServerID, 10, 64); err != nil {
		return fmt.Errorf("file_server_id must be numeric: %q", fileServerID)
	}

	detail, err := ReadFileServerDetail(c, ctx, fileServerID)
	if err != nil {
		if errors.Is(err, ErrFileServerNotFound) {
			return fmt.Errorf("file server id %s not found", fileServerID)
		}
		return fmt.Errorf("failed to validate file server id %s: %w", fileServerID, err)
	}
	if detail.EngagementID == 0 {
		return fmt.Errorf("file server id %s has no engagement_id in platform metadata", fileServerID)
	}
	if detail.EndpointID == 0 {
		return fmt.Errorf("file server id %s has no endpoint_id in platform metadata", fileServerID)
	}
	return ValidateEngagementAndEndpoint(c, ctx, detail.EngagementID, detail.EndpointID)
}

// ValidateNASOrderForFileServerCreate ensures a NAS base order exists for a new file server (P2R engagements).
func ValidateNASOrderForFileServerCreate(c *client.Client, ctx context.Context, engagementID, endpointID int64, fileStorageType string) error {
	if err := ValidateEngagementAndEndpoint(c, ctx, engagementID, endpointID); err != nil {
		return err
	}
	return nas.ValidateNASOrder(c, ctx, engagementID, endpointID, fileStorageType)
}
