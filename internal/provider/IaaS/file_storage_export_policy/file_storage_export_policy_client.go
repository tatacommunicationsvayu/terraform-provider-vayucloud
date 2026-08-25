// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_export_policy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

const (
	attachClientPath = "/attachClient"
	detachClientPath = "/volumes/%d/clients/%s"
)

// NasClientActionRequest is the JSON body for attachClient (POST).
type NasClientActionRequest struct {
	IP       string `json:"ip"`
	Name     string `json:"name"`
	NasVolCi int64  `json:"nasVolCi"`
}

func buildNasClientActionRequest(clientIP, volumeName, nasVolCiID string) (*NasClientActionRequest, error) {
	nasVolCi, err := strconv.ParseInt(nasVolCiID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("file_storage_volume_id %q is not a valid nasVolCi: %w", nasVolCiID, err)
	}
	return &NasClientActionRequest{
		IP:       clientIP,
		Name:     volumeName,
		NasVolCi: nasVolCi,
	}, nil
}

func nasClientAction(c *client.Client, ctx context.Context, method, path string, req *NasClientActionRequest) (*client.AuditResponse, error) {
	fullPath := common.FileStorageServicePath + path
	respBody, err := c.DoRequest(ctx, method, fullPath, req)
	if err != nil {
		return nil, err
	}
	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse NAS client action response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("NAS client action failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	tflog.Info(ctx, "NAS client action initiated", map[string]any{"audit_id": result.Data.Audit.AuditID, "path": path, "method": method})
	return &result, nil
}

// AttachNASClient attaches a client IP to a NAS volume (POST .../nas/attachClient).
func AttachNASClient(c *client.Client, ctx context.Context, clientIP, volumeName, nasVolCiID string) (*client.AuditResponse, error) {
	req, err := buildNasClientActionRequest(clientIP, volumeName, nasVolCiID)
	if err != nil {
		return nil, err
	}
	resp, err := nasClientAction(c, ctx, http.MethodPost, attachClientPath, req)
	if err != nil {
		return nil, fmt.Errorf("attach NAS client: %w", err)
	}
	return resp, nil
}

// AttachNASClientAndWait attaches a client and waits for audit completion (no action-state; attachClient is not registered there).
// Retries when the portal reports the volume or export policy is not ready yet.
func AttachNASClientAndWait(c *client.Client, ctx context.Context, clientIP, volumeName, nasVolCiID string) (*client.AuditLogResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= client.NasReadinessPollMaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := AttachNASClient(c, ctx, clientIP, volumeName, nasVolCiID)
		if err != nil {
			lastErr = err
			if !isTransientNASAttachError(err) || attempt == client.NasReadinessPollMaxAttempts {
				return nil, err
			}
			tflog.Debug(ctx, "NAS attach client not ready yet, retrying", map[string]any{
				"nas_vol_ci": nasVolCiID,
				"attempt":    attempt,
				"error":      err.Error(),
			})
			if waitErr := waitNasReadiness(ctx); waitErr != nil {
				return nil, waitErr
			}
			continue
		}

		auditLog, err := c.WaitForAuditCompletionNoActionState(ctx, resp.Data.Audit.AuditID)
		if err == nil {
			return auditLog, nil
		}
		lastErr = err
		if !isTransientNASAttachError(err) || attempt == client.NasReadinessPollMaxAttempts {
			return nil, err
		}
		tflog.Debug(ctx, "NAS attach client audit not ready yet, retrying", map[string]any{
			"nas_vol_ci": nasVolCiID,
			"attempt":    attempt,
			"error":      err.Error(),
		})
		if waitErr := waitNasReadiness(ctx); waitErr != nil {
			return nil, waitErr
		}
	}

	return nil, fmt.Errorf("attach NAS client: timed out after %d attempts: %w", client.NasReadinessPollMaxAttempts, lastErr)
}

func isTransientNASAttachError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no policyid found") ||
		strings.Contains(msg, "no matching nas volume") ||
		strings.Contains(msg, "selected nas svm is not found")
}

func waitNasReadiness(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(client.AuditPollInterval):
		return nil
	}
}

// DetachNASClient detaches a client IP via DELETE .../nas/volumes/{volumeId}/clients/{clientIp}.
// clientIp is path-escaped (e.g. CIDR slash becomes %2F).
func DetachNASClient(c *client.Client, ctx context.Context, clientIP, nasVolCiID string) (*client.AuditResponse, error) {
	nasVolCi, err := strconv.ParseInt(nasVolCiID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("file_storage_volume_id %q is not a valid volumeId: %w", nasVolCiID, err)
	}
	path := fmt.Sprintf(detachClientPath, nasVolCi, url.PathEscape(strings.TrimSpace(clientIP)))
	fullPath := common.FileStorageServicePath + path
	respBody, err := c.DoRequest(ctx, http.MethodDelete, fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("detach NAS client: %w", err)
	}
	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse NAS client detach response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("NAS client detach failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	tflog.Info(ctx, "NAS client detach initiated", map[string]any{"audit_id": result.Data.Audit.AuditID, "path": path})
	return &result, nil
}

// DetachNASClientAndWait detaches a client and waits for audit completion (no action-state).
func DetachNASClientAndWait(c *client.Client, ctx context.Context, clientIP, nasVolCiID string) (*client.AuditLogResponse, error) {
	resp, err := DetachNASClient(c, ctx, clientIP, nasVolCiID)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletionNoActionState(ctx, resp.Data.Audit.AuditID)
}
