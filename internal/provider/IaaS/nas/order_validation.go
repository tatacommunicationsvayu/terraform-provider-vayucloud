// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

// Package nas provides shared NAS API helpers used by file_server and file_storage_volume resources.
package nas

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

const validateNASOrderPath = "/validateNASOrder"

// ValidateNASOrder checks that a NAS base order exists for the engagement, endpoint, and file storage type.
// For P2R engagements the portal requires a matching available NAS order line before volume create.
func ValidateNASOrder(c *client.Client, ctx context.Context, engagementID, endpointID int64, fileStorageType string) error {
	if fileStorageType == "" {
		return fmt.Errorf("file_storage_type is required for NAS order validation")
	}
	normalizedType := NormalizeFileStorageTypeForOrder(fileStorageType)
	path := fmt.Sprintf(
		"%s%s?engagementId=%d&endpointId=%d&fileStorageType=%s",
		common.FileStorageServicePath,
		validateNASOrderPath,
		engagementID,
		endpointID,
		url.QueryEscape(normalizedType),
	)
	tflog.Debug(ctx, "ValidateNASOrder", map[string]any{
		"method": http.MethodPost,
		"path":   path,
	})

	statusCode, respBody, err := c.DoRequestWithHTTPStatus(ctx, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("validate NAS order: %w", err)
	}

	env, err := parseCatalystEnvelope(respBody)
	if err != nil {
		return fmt.Errorf("parse validate NAS order response (HTTP %d): %w", statusCode, err)
	}

	if statusCode >= 200 && statusCode < 300 && strings.EqualFold(strings.TrimSpace(env.Status), "success") {
		return nil
	}

	return fmt.Errorf("%s", catalystFailureMessage(env))
}
