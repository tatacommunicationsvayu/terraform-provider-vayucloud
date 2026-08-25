// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

// WaitForNASVserverReady polls until the vserver is visible in getNASVservers for the engagement and endpoint.
// This mirrors portal preCheckNASCreation linkage so createNASVolume does not fail with "Selected NAS SVM is not found."
func WaitForNASVserverReady(c *client.Client, ctx context.Context, engagementID, endpointID int64, vserverID string) error {
	tflog.Info(ctx, "Waiting for NAS vserver readiness", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
		"vserver_id":    vserverID,
		"max_attempts":  client.NasReadinessPollMaxAttempts,
	})

	var lastErr error
	for attempt := 1; attempt <= client.NasReadinessPollMaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := ValidateFileServerInEngagement(c, ctx, engagementID, endpointID, vserverID); err == nil {
			tflog.Info(ctx, "NAS vserver is ready for volume operations", map[string]any{
				"engagement_id": engagementID,
				"endpoint_id":   endpointID,
				"vserver_id":    vserverID,
				"attempts":      attempt,
			})
			return nil
		} else {
			lastErr = err
		}

		if attempt < client.NasReadinessPollMaxAttempts {
			tflog.Debug(ctx, "NAS vserver not ready yet, retrying", map[string]any{
				"engagement_id": engagementID,
				"endpoint_id":   endpointID,
				"vserver_id":    vserverID,
				"attempt":       attempt,
				"error":         lastErr.Error(),
			})
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(client.AuditPollInterval):
			}
		}
	}

	return fmt.Errorf(
		"NAS vserver %s not ready for engagement %d endpoint %d after %d attempts (%v): %w",
		vserverID, engagementID, endpointID, client.NasReadinessPollMaxAttempts,
		time.Duration(client.NasReadinessPollMaxAttempts)*client.AuditPollInterval,
		lastErr,
	)
}
