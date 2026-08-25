// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/file_server"
)

const (
	volumeCreatePath       = "/createNASVolume/%d"
	volumeDeletePath       = "/volumes/%d"
	volumeResizePath       = "/resizeNasVolume"
	volumeDetailsPath      = "/fetchVolumeDetails/%d/%s/%s"
	volumePrecheckSizePath = "/precheckNasVolumeSize/%d"
)

// FileStorageVolumeCreateRequest is the POST body for createNASVolume (CreateNasVolumeVO).
type FileStorageVolumeCreateRequest struct {
	EndpointID      int64  `json:"endpointId"`
	EngagementID    int64  `json:"engagementId"`
	VolumeName      string `json:"volumeName"`
	VolumeSize      int64  `json:"volumeSize"`
	VolumeType      string `json:"volumeType"`
	VserverID       int64  `json:"vserverId"`
	FileStorageType string `json:"fileStorageType,omitempty"`
	UsageType       string `json:"usageType,omitempty"`
	PricingModel    string `json:"pricingModel,omitempty"`
	Iops            int    `json:"iops,omitempty"`
}

// FileStorageVolumeResizeRequest is the PUT body for resizeNasVolume.
// Preferred API shape: { "volumeId": <int>, "volumeSize": <int> }
type FileStorageVolumeResizeRequest struct {
	VolumeID   int64  `json:"volumeId"`
	VolumeSize int64  `json:"volumeSize"`
	Unit       string `json:"unit,omitempty"`
}

// FileStorageVolumeDetail is a normalized volume detail for Read refresh.
type FileStorageVolumeDetail struct {
	Name   string
	SizeGb int64
}

type volumeEnvelope struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	Data         json.RawMessage `json:"data"`
	ResponseCode int             `json:"responseCode"`
}

func doNasVolumeAction(c *client.Client, ctx context.Context, method, path string, body any) (string, error) {
	fullPath := common.FileStorageServicePath + path
	respBody, err := c.DoRequest(ctx, method, fullPath, body)
	if err != nil {
		return "", err
	}
	var env volumeEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return "", fmt.Errorf("parse NAS volume response: %w", err)
	}
	if env.Status != "" && env.Status != "success" {
		if env.Message != "" {
			return "", fmt.Errorf("NAS volume API failed: %s (code: %d)", env.Message, env.ResponseCode)
		}
		return "", fmt.Errorf("NAS volume API failed (code: %d)", env.ResponseCode)
	}
	auditID, err := extractAuditIDFromNasResponse(respBody, env.Data)
	if err != nil {
		return "", err
	}
	tflog.Info(ctx, "NAS volume action initiated", map[string]any{"audit_id": auditID, "path": path, "method": method})
	return auditID, nil
}

func postNasVolumeAction(c *client.Client, ctx context.Context, path string, body any) (string, error) {
	return doNasVolumeAction(c, ctx, http.MethodPost, path, body)
}

func extractAuditIDFromNasResponse(respBody, data json.RawMessage) (string, error) {
	var standard client.AuditResponse
	if err := json.Unmarshal(respBody, &standard); err == nil {
		if id := strings.TrimSpace(standard.Data.Audit.AuditID); id != "" {
			return id, nil
		}
	}
	if len(data) > 0 {
		var nested struct {
			AuditID string `json:"auditID"`
			AuditId string `json:"auditId"`
			Audit   struct {
				AuditID string `json:"auditId"`
			} `json:"audit"`
		}
		if err := json.Unmarshal(data, &nested); err == nil {
			if id := firstNonEmpty(nested.AuditID, nested.AuditId, nested.Audit.AuditID); id != "" {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("audit id not found in NAS volume response")
}

// CreateFileStorageVolume starts NAS volume creation via POST .../nas/createNASVolume/{engagementId}.
func CreateFileStorageVolume(c *client.Client, ctx context.Context, req *FileStorageVolumeCreateRequest) (string, error) {
	path := fmt.Sprintf(volumeCreatePath, req.EngagementID)
	return postNasVolumeAction(c, ctx, path, req)
}

// CreateFileStorageVolumeAndWait waits for the vserver to be ready, creates a volume, and waits for audit completion.
func CreateFileStorageVolumeAndWait(c *client.Client, ctx context.Context, req *FileStorageVolumeCreateRequest) (*client.AuditLogResponse, error) {
	vserverID := strconv.FormatInt(req.VserverID, 10)
	if err := file_server.WaitForNASVserverReady(c, ctx, req.EngagementID, req.EndpointID, vserverID); err != nil {
		return nil, fmt.Errorf("create file storage volume: %w", err)
	}
	auditID, err := CreateFileStorageVolume(c, ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create file storage volume: %w", err)
	}
	return c.WaitForAuditCompletionNoActionState(ctx, auditID)
}

// ResolveNasVolumeCIAfterCreate polls fetchVolumeDetails until volCi (ci_master id) is available.
// attachClient expects nasVolCi = ci_master.id; audit resourceId is often a different identifier.
func ResolveNasVolumeCIAfterCreate(c *client.Client, ctx context.Context, engagementID int64, fileServerID, volumeName string) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= client.NasReadinessPollMaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		lookup, err := GetFileStorageVolumeDetails(c, ctx, engagementID, fileServerID, volumeName)
		if err == nil && strings.TrimSpace(lookup.VolumeID) != "" {
			tflog.Info(ctx, "Resolved NAS volume CI (nasVolCi) after create", map[string]any{
				"nas_vol_ci": lookup.VolumeID,
				"volume":     volumeName,
				"attempt":    attempt,
			})
			return lookup.VolumeID, nil
		}
		lastErr = err
		if err == nil {
			lastErr = fmt.Errorf("fetchVolumeDetails returned empty volCi for volume %q", volumeName)
		}
		tflog.Debug(ctx, "NAS volume CI not ready yet, retrying fetchVolumeDetails", map[string]any{
			"volume":  volumeName,
			"attempt": attempt,
			"error":   lastErr.Error(),
		})
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(client.AuditPollInterval):
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("volCi not available")
	}
	return "", fmt.Errorf("resolve nasVolCi after create: timed out after %d attempts: %w", client.NasReadinessPollMaxAttempts, lastErr)
}

// NasVolumeSizePrecheck is the portal payload from GET precheckNasVolumeSize.
// Portal allows resize only when requestedSize > currentSize (no degrade/shrink).
type NasVolumeSizePrecheck struct {
	VolumeID      int64
	IsAllowed     bool
	CurrentSize   *int64
	RequestedSize int64
	PortalError   string
}

// ErrNasVolumeResizeNotAllowed indicates the portal rejected a resize (shrink, equal size, or unknown current size).
var ErrNasVolumeResizeNotAllowed = errors.New("NAS volume resize not allowed")

func (p *NasVolumeSizePrecheck) disallowError() error {
	if strings.TrimSpace(p.PortalError) != "" {
		return fmt.Errorf("%w: %s", ErrNasVolumeResizeNotAllowed, strings.TrimSpace(p.PortalError))
	}
	if p.CurrentSize == nil {
		return fmt.Errorf(
			"%w: volume %d — portal could not determine current size (requested %d GB)",
			ErrNasVolumeResizeNotAllowed, p.VolumeID, p.RequestedSize,
		)
	}
	current := *p.CurrentSize
	if p.RequestedSize <= current {
		return fmt.Errorf(
			"%w: requested size %d GB must be greater than current portal size %d GB (degrade/shrink not supported); run terraform refresh if the volume was resized outside Terraform",
			ErrNasVolumeResizeNotAllowed, p.RequestedSize, current,
		)
	}
	return fmt.Errorf(
		"%w: volume %d resize to %d GB rejected by portal precheck (current %d GB)",
		ErrNasVolumeResizeNotAllowed, p.VolumeID, p.RequestedSize, current,
	)
}

type nasVolumeSizePrecheckData struct {
	IsAllowed     bool   `json:"isAllowed"`
	CurrentSize   *int64 `json:"currentSize"`
	RequestedSize int64  `json:"requestedSize"`
	Error         string `json:"error"`
}

func parseNasVolumeSizePrecheckResponse(volumeID int64, respBody []byte) (*NasVolumeSizePrecheck, error) {
	var env volumeEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("parse precheckNasVolumeSize envelope: %w", err)
	}
	st := strings.TrimSpace(strings.ToLower(env.Status))
	if st != "" && st != "success" {
		out := &NasVolumeSizePrecheck{VolumeID: volumeID}
		if len(env.Data) > 0 && string(env.Data) != "null" {
			var data nasVolumeSizePrecheckData
			if err := json.Unmarshal(env.Data, &data); err == nil {
				out.PortalError = firstNonEmpty(data.Error, env.Message)
				out.RequestedSize = data.RequestedSize
				out.CurrentSize = data.CurrentSize
				out.IsAllowed = data.IsAllowed
			}
		}
		if out.PortalError == "" {
			out.PortalError = firstNonEmpty(env.Message, "precheckNasVolumeSize failed")
		}
		return nil, fmt.Errorf("precheckNasVolumeSize: %s (responseCode=%d)", out.PortalError, env.ResponseCode)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil, fmt.Errorf("precheckNasVolumeSize: empty response data for volume %d", volumeID)
	}
	var data nasVolumeSizePrecheckData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("parse precheckNasVolumeSize data: %w", err)
	}
	return &NasVolumeSizePrecheck{
		VolumeID:      volumeID,
		IsAllowed:     data.IsAllowed,
		CurrentSize:   data.CurrentSize,
		RequestedSize: data.RequestedSize,
		PortalError:   strings.TrimSpace(data.Error),
	}, nil
}

// PrecheckNasVolumeSize calls GET .../nas/precheckNasVolumeSize/{volumeId}?requestedSize=.
func PrecheckNasVolumeSize(c *client.Client, ctx context.Context, volumeID, requestedSizeGB int64) (*NasVolumeSizePrecheck, error) {
	if volumeID < 1 {
		return nil, fmt.Errorf("precheckNasVolumeSize: volume id must be positive")
	}
	if requestedSizeGB < 1 {
		return nil, fmt.Errorf("precheckNasVolumeSize: requested size must be at least 1 GB")
	}
	path := fmt.Sprintf("%s"+volumePrecheckSizePath+"?requestedSize=%d",
		common.FileStorageServicePath, volumeID, requestedSizeGB)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("precheckNasVolumeSize: %w", err)
	}
	out, err := parseNasVolumeSizePrecheckResponse(volumeID, respBody)
	if err != nil {
		return nil, err
	}
	tflog.Info(ctx, "NAS volume size precheck", map[string]any{
		"volume_id":      volumeID,
		"requested_size": out.RequestedSize,
		"current_size":   out.CurrentSize,
		"is_allowed":     out.IsAllowed,
	})
	return out, nil
}

// ResizeFileStorageVolume starts a volume resize via PUT .../nas/resizeNasVolume.
func ResizeFileStorageVolume(c *client.Client, ctx context.Context, req *FileStorageVolumeResizeRequest) (string, error) {
	return doNasVolumeAction(c, ctx, http.MethodPut, volumeResizePath, req)
}

// ResizeFileStorageVolumeAndWait prechecks portal size rules, then resizes and waits for audit completion.
func ResizeFileStorageVolumeAndWait(c *client.Client, ctx context.Context, req *FileStorageVolumeResizeRequest) (*client.AuditLogResponse, error) {
	precheck, err := PrecheckNasVolumeSize(c, ctx, req.VolumeID, req.VolumeSize)
	if err != nil {
		return nil, fmt.Errorf("resize file storage volume: %w", err)
	}
	if !precheck.IsAllowed {
		return nil, precheck.disallowError()
	}
	auditID, err := ResizeFileStorageVolume(c, ctx, req)
	if err != nil {
		return nil, fmt.Errorf("resize file storage volume: %w", err)
	}
	return c.WaitForAuditCompletionNoActionState(ctx, auditID)
}

// DeleteFileStorageVolume deletes a volume via DELETE .../nas/volumes/{volumeId} (no body).
func DeleteFileStorageVolume(c *client.Client, ctx context.Context, volumeID int64) (string, error) {
	path := fmt.Sprintf(volumeDeletePath, volumeID)
	return doNasVolumeAction(c, ctx, http.MethodDelete, path, nil)
}

// DeleteFileStorageVolumeAndWait deletes and waits for audit completion.
func DeleteFileStorageVolumeAndWait(c *client.Client, ctx context.Context, volumeID int64) (*client.AuditLogResponse, error) {
	auditID, err := DeleteFileStorageVolume(c, ctx, volumeID)
	if err != nil {
		return nil, fmt.Errorf("delete file storage volume: %w", err)
	}
	return c.WaitForAuditCompletionNoActionState(ctx, auditID)
}

// ParseVserverID converts file_server_id to the numeric vserverId for NAS APIs.
func ParseVserverID(fileServerID string) (int64, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(fileServerID), 10, 64)
	if err != nil || v < 1 {
		return 0, fmt.Errorf("file_server_id %q must be a positive integer vserver id", fileServerID)
	}
	return v, nil
}

// ErrFileStorageVolumeNotFound indicates the volume no longer exists.
var ErrFileStorageVolumeNotFound = errors.New("file storage volume not found")

// FileStorageVolumeLookupDetail is a normalized subset of fetchVolumeDetails response.
type FileStorageVolumeLookupDetail struct {
	VolumeID           string
	Name               string
	VserverDisplayName string
	FileServerID       string
	Clients            []string
	SizeGb             int64
	RawBody            string
}

// GetFileStorageVolumeDetails calls GET .../nas/fetchVolumeDetails/{engagementId}/{fileServerId}/{volumeName}.
func GetFileStorageVolumeDetails(c *client.Client, ctx context.Context, engagementID int64, fileServerID, storageName string) (*FileStorageVolumeLookupDetail, error) {
	path := fmt.Sprintf(common.FileStorageServicePath+volumeDetailsPath,
		engagementID,
		url.PathEscape(fileServerID),
		url.PathEscape(storageName),
	)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: engagement %d file_server %s name %q", ErrFileStorageVolumeNotFound, engagementID, fileServerID, storageName)
		}
		return nil, err
	}
	rawStr := string(respBody)
	var env volumeEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("parse fetchVolumeDetails envelope: %w", err)
	}
	if env.Status != "" && env.Status != "success" {
		return nil, fmt.Errorf("fetchVolumeDetails failed: %s (code: %d)", env.Message, env.ResponseCode)
	}
	out := &FileStorageVolumeLookupDetail{RawBody: rawStr}
	if len(env.Data) == 0 {
		if err := populateVolumeLookupFromJSON(respBody, out); err != nil {
			return nil, err
		}
		return out, nil
	}
	if err := populateVolumeLookupFromJSON(env.Data, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetFileStorageVolume loads volume name and size via fetchVolumeDetails.
func GetFileStorageVolume(c *client.Client, ctx context.Context, engagementID int64, fileServerID, volumeName string) (*FileStorageVolumeDetail, error) {
	lookup, err := GetFileStorageVolumeDetails(c, ctx, engagementID, fileServerID, volumeName)
	if err != nil {
		return nil, err
	}
	return &FileStorageVolumeDetail{
		Name:   firstNonEmpty(lookup.Name, volumeName),
		SizeGb: lookup.SizeGb,
	}, nil
}

func populateVolumeLookupFromJSON(raw []byte, out *FileStorageVolumeLookupDetail) error {
	var extra struct {
		ID                 any      `json:"id"`
		VolumeID           any      `json:"volumeId"`
		VolCi              any      `json:"volCi"`
		VolumeName         string   `json:"volumeName"`
		VserverDisplayName string   `json:"vserverDisplayName"`
		VserverID          any      `json:"vserverId"`
		SizeGb             any      `json:"sizeGb"`
		VolumeSize         any      `json:"volumeSize"`
		Size               any      `json:"size"`
		Clients            []string `json:"clients"`
	}
	if err := json.Unmarshal(raw, &extra); err != nil {
		return err
	}
	out.VolumeID = firstNonEmpty(
		jsonScalarToString(extra.VolumeID),
		jsonScalarToString(extra.VolCi),
		jsonScalarToString(extra.ID),
	)
	if extra.VolumeName != "" {
		out.Name = extra.VolumeName
	}
	if extra.VserverDisplayName != "" {
		out.VserverDisplayName = extra.VserverDisplayName
	}
	if extra.VserverID != nil {
		out.FileServerID = jsonScalarToString(extra.VserverID)
	}
	if extra.SizeGb != nil {
		out.SizeGb = parseNasVolumeSizeGB(extra.SizeGb)
	}
	if out.SizeGb == 0 && extra.VolumeSize != nil {
		out.SizeGb = parseNasVolumeSizeGB(extra.VolumeSize)
	}
	if out.SizeGb == 0 && extra.Size != nil {
		out.SizeGb = parseNasVolumeSizeGB(extra.Size)
	}
	if len(extra.Clients) > 0 {
		out.Clients = append([]string(nil), extra.Clients...)
	}
	return nil
}

func jsonScalarToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return strings.TrimSpace(t.String())
	case bool:
		return strconv.FormatBool(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func firstNonEmpty(vals ...string) string {
	for _, s := range vals {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
