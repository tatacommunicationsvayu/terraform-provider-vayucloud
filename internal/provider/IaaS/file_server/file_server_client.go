// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// Audit identifiers for WaitForAuditCompletion — align with backend action-state module/action names.
const (
	fileServerAuditModule       = "FileServer"
	fileServerAuditActionCreate = "create"
	fileServerAuditActionRead   = "read"
	fileServerAuditActionDelete = "delete"
)

// HTTP route suffixes under FileStorageServicePath — adjust when API paths are finalized.
const (
	// createNasVserverPath is POST .../nas/createNasVserver/{engagementId} (engagementId matches JSON body).
	fileServerCreateNasVserverPath = "/createNasVserver/%d"
	// deleteNasVserverPath is DELETE .../nas/vservers/{vserverId} (no body).
	fileServerDeleteNasVserverPath = "/vservers/%d"
	// listNASVserversPath is GET .../nas/getNASVservers/{engagementId}/{endpointId}
	fileServerListNASVserversPath = "/getNASVservers/%d/%d"
	// fetchVserverDetailsPath is GET .../nas/fetchVserverDetails/{vserverId} (single vserver detail).
	fileServerFetchVserverDetailsPath = "/fetchVserverDetails/%s"
)

// FileServerCreateRequest is the POST body for createNasVserver (path includes engagementId).
type FileServerCreateRequest struct {
	EndpointID      int64  `json:"endpointId"`
	EngagementID    int64  `json:"engagementId"`
	VserverName     string `json:"vserverName"`
	FileStorageType string `json:"fileStorageType"`
}

// FileServerDetail is a GET response payload subset used for Read refresh.
// fileServerName is the display name users configure as vserver_name. vserverName/name
// may be the platform's generated component name, so expose them only as fallback.
type FileServerDetail struct {
	Name            string `json:"name,omitempty"`
	VserverName     string `json:"vserverName,omitempty"`
	FileServerName  string `json:"fileServerName,omitempty"`
	EndpointID      int64  `json:"endpointId"`
	EngagementID    int64  `json:"engagementId"`
	FileStorageType string `json:"fileStorageType,omitempty"`
	FileServerType  string `json:"fileServerType,omitempty"`
	Description     string `json:"description,omitempty"`
}

// ErrFileServerNotFound is returned when the platform has no file server for the given id.
var ErrFileServerNotFound = errors.New("file server not found")

func normalizeFileServerDetail(d *FileServerDetail) {
	if d.FileServerName == "" {
		if d.VserverName != "" {
			d.FileServerName = d.VserverName
		} else if d.Name != "" {
			d.FileServerName = d.Name
		}
	}
	if d.FileStorageType == "" && d.FileServerType != "" {
		d.FileStorageType = canonicalFileServerStorageType(d.FileServerType)
	} else if d.FileStorageType != "" {
		d.FileStorageType = canonicalFileServerStorageType(d.FileStorageType)
	}
}

// FileServerActionStateReadData is the `data` object from action-state read (module=FileServer).
type FileServerActionStateReadData struct {
	ResourceID      int64  `json:"resourceId"`
	EngagementID    int64  `json:"engagementId"`
	EndpointID      int64  `json:"endpointId"`
	VserverName     string `json:"vserverName"`
	FileServerName  string `json:"fileServerName"`
	Name            string `json:"name"`
	FileStorageType string `json:"fileStorageType"`
	FileServerType  string `json:"fileServerType"`
	Description     string `json:"description,omitempty"`
}

func fileServerReadDataToDetail(d *FileServerActionStateReadData) FileServerDetail {
	out := FileServerDetail{
		EndpointID:      d.EndpointID,
		EngagementID:    d.EngagementID,
		VserverName:     d.VserverName,
		FileServerName:  d.FileServerName,
		Name:            d.Name,
		FileStorageType: d.FileStorageType,
		FileServerType:  d.FileServerType,
		Description:     d.Description,
	}
	normalizeFileServerDetail(&out)
	return out
}

type fileServerEnvelope struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	Data         json.RawMessage `json:"data"`
	ResponseCode int             `json:"responseCode"`
}

// NASVserverListEntry is one vserver normalized from getNASVservers.
type NASVserverListEntry struct {
	VserverID       int64
	VserverName     string
	EngagementID    int64
	EndpointID      int64
	FileStorageType string
	Description     string
}

// NASVserverListResponse is the parsed getNASVservers API response.
type NASVserverListResponse struct {
	Items        []NASVserverListEntry
	Status       string
	Message      string
	ResponseCode int
	RawBody      string
}

type nasVserverListRow struct {
	VserverID       json.Number `json:"vserverId"`
	VserverName     string      `json:"vserverName"`
	EngagementID    int64       `json:"engagementId"`
	EndpointID      int64       `json:"endpointId"`
	FileStorageType string      `json:"fileStorageType"`
	Description     string      `json:"description,omitempty"`
	Name            string      `json:"name"`
	ID              json.Number `json:"id"`
}

func (r nasVserverListRow) toEntry() (NASVserverListEntry, error) {
	var vid int64
	var err error
	switch {
	case r.VserverID != "":
		vid, err = r.VserverID.Int64()
	case r.ID != "":
		vid, err = r.ID.Int64()
	default:
		return NASVserverListEntry{}, fmt.Errorf("missing vserver id in list row")
	}
	if err != nil {
		return NASVserverListEntry{}, err
	}
	name := r.VserverName
	if name == "" {
		name = r.Name
	}
	return NASVserverListEntry{
		VserverID:       vid,
		VserverName:     name,
		EngagementID:    r.EngagementID,
		EndpointID:      r.EndpointID,
		FileStorageType: r.FileStorageType,
		Description:     r.Description,
	}, nil
}

func parseNASVserverListRows(data json.RawMessage) ([]nasVserverListRow, error) {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" {
		return nil, nil
	}
	var rows []nasVserverListRow
	if err := json.Unmarshal(data, &rows); err == nil {
		return rows, nil
	}
	// Portal getNASVservers returns endpoint-keyed buckets, e.g.
	// {"EP_V2_BL":[{vserverId:57119,...}],"EP_V2_DEL":[...]} — merge all buckets.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse NAS vserver list data: %w", err)
	}
	var all []nasVserverListRow
	for _, raw := range m {
		var bucket []nasVserverListRow
		if err := json.Unmarshal(raw, &bucket); err != nil {
			continue
		}
		all = append(all, bucket...)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("unrecognized NAS vserver list payload shape: %s", s)
	}
	return all, nil
}

// ListNASVservers returns NAS vservers for an engagement and endpoint.
// GET {FileStorageServicePath}/getNASVservers/{engagementId}/{endpointId}
func ListNASVservers(c *client.Client, ctx context.Context, engagementID, endpointID int64) (*NASVserverListResponse, error) {
	path := fmt.Sprintf(common.FileStorageServicePath+fileServerListNASVserversPath, engagementID, endpointID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list NAS vservers: %w", err)
	}
	out := &NASVserverListResponse{RawBody: string(respBody)}

	trim := strings.TrimSpace(string(respBody))
	if strings.HasPrefix(trim, "[") {
		var rows []nasVserverListRow
		if err := json.Unmarshal(respBody, &rows); err != nil {
			return nil, fmt.Errorf("parse getNASVservers array: %w", err)
		}
		out.Status = "success"
		for _, row := range rows {
			e, err := row.toEntry()
			if err != nil {
				continue
			}
			out.Items = append(out.Items, e)
		}
		return out, nil
	}

	var env fileServerEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("parse getNASVservers response: %w", err)
	}
	if env.Status != "" && env.Status != "success" {
		return nil, fmt.Errorf("getNASVservers failed: %s (code: %d)", env.Message, env.ResponseCode)
	}
	out.Status = env.Status
	out.Message = env.Message
	out.ResponseCode = env.ResponseCode
	rows, err := parseNASVserverListRows(env.Data)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		e, err := row.toEntry()
		if err != nil {
			continue
		}
		out.Items = append(out.Items, e)
	}
	tflog.Info(ctx, "Listed NAS vservers", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
		"count":         len(out.Items),
	})
	return out, nil
}

// CreateFileServer starts NAS vserver creation (async audit flow).
// POST {FileStorageServicePath}/createNasVserver/{engagementId}
func CreateFileServer(c *client.Client, ctx context.Context, req *FileServerCreateRequest) (*client.AuditResponse, error) {
	path := fmt.Sprintf(common.FileStorageServicePath+fileServerCreateNasVserverPath, req.EngagementID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("create file server: %w", err)
	}
	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse create file server response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("create file server failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	tflog.Info(ctx, "File server create initiated", map[string]any{"audit_id": result.Data.Audit.AuditID})
	return &result, nil
}

// CreateFileServerAndWait creates a file server and waits for audit completion.
func CreateFileServerAndWait(c *client.Client, ctx context.Context, req *FileServerCreateRequest) (*client.AuditLogResponse, error) {
	resp, err := CreateFileServer(c, ctx, req)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletion(ctx, resp.Data.Audit.AuditID, fileServerAuditActionCreate, fileServerAuditModule, map[string]any{
		"vserverName":     req.VserverName,
		"engagementId":    req.EngagementID,
		"endpointId":      req.EndpointID,
		"fileStorageType": req.FileStorageType,
	})
}

// DeleteFileServer starts NAS vserver deletion (async audit flow).
// DELETE {FileStorageServicePath}/vservers/{vserverId} (no body).
func DeleteFileServer(c *client.Client, ctx context.Context, vserverID int64) (*client.AuditResponse, error) {
	path := fmt.Sprintf(common.FileStorageServicePath+fileServerDeleteNasVserverPath, vserverID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("delete file server: %w", err)
	}
	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse delete file server response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("delete file server failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	tflog.Info(ctx, "File server delete initiated", map[string]any{"audit_id": result.Data.Audit.AuditID, "vserver_id": vserverID})
	return &result, nil
}

// DeleteFileServerAndWait deletes and waits for audit completion.
func DeleteFileServerAndWait(c *client.Client, ctx context.Context, vserverID int64) (*client.AuditLogResponse, error) {
	resp, err := DeleteFileServer(c, ctx, vserverID)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletion(ctx, resp.Data.Audit.AuditID, fileServerAuditActionDelete, fileServerAuditModule, map[string]any{
		"vserverId": vserverID,
	})
}

// ReadFileServer reads file server state.
//
// GET {FileStorageServicePath}/fileserver-state/{fileServerId}
// Returns the same response envelope as the legacy action-state read API (module=FileServer, action=read).
func ReadFileServer(c *client.Client, ctx context.Context, vserverID string) (*common.ActionStateResponse, *FileServerDetail, error) {
	tflog.Debug(ctx, "Reading file server state", map[string]any{"resource_id": vserverID})

	vid, err := strconv.ParseInt(vserverID, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("file server id must be numeric for state read: %w", err)
	}

	path := fmt.Sprintf("%s/fileserver-state/%d", common.FileStorageServicePath, vid)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file server: %w", err)
	}

	tflog.Debug(ctx, "File server state response", map[string]any{
		"response": string(respBody),
	})

	var actionResp common.ActionStateResponse
	if err := json.Unmarshal(respBody, &actionResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse file server state response: %w", err)
	}

	if actionResp.Status != "success" {
		msg := strings.ToLower(actionResp.Message)
		if strings.Contains(msg, "not found") {
			return &actionResp, nil, fmt.Errorf("%w: %s", ErrFileServerNotFound, vserverID)
		}
		return &actionResp, nil, fmt.Errorf("file server read failed: %s (code: %d)", actionResp.Message, actionResp.ResponseCode)
	}

	var readData FileServerActionStateReadData
	if len(actionResp.Data) > 0 {
		if err := json.Unmarshal(actionResp.Data, &readData); err != nil {
			return &actionResp, nil, fmt.Errorf("failed to parse file server read data: %w", err)
		}
	}

	detail := fileServerReadDataToDetail(&readData)
	tflog.Info(ctx, "File server read successfully", map[string]any{"resource_id": vid})

	return &actionResp, &detail, nil
}

// ReadFileServerDetail loads file server attributes for refresh and delete. It prefers fileserver-state
// and falls back to GET fetchVserverDetails when state read fails,
// so state can still store engagement_id for outputs and destroy.
func ReadFileServerDetail(c *client.Client, ctx context.Context, vserverID string) (*FileServerDetail, error) {
	_, detail, err := ReadFileServer(c, ctx, vserverID)
	if err == nil && detail != nil && detail.EngagementID != 0 && detail.EndpointID != 0 {
		return detail, nil
	}
	if errors.Is(err, ErrFileServerNotFound) {
		return nil, err
	}
	fallback, gerr := GetFileServer(c, ctx, vserverID)
	if gerr != nil {
		if errors.Is(gerr, ErrFileServerNotFound) {
			return nil, gerr
		}
		return nil, fmt.Errorf("file server read failed (fileserver-state: %v; fetchVserverDetails: %w)", err, gerr)
	}
	tflog.Info(ctx, "File server detail loaded via fetchVserverDetails after fileserver-state read failure")
	return fallback, nil
}

// GetFileServer loads current file server state from the API via fetchVserverDetails (direct GET).
func GetFileServer(c *client.Client, ctx context.Context, id string) (*FileServerDetail, error) {
	path := fmt.Sprintf(common.FileStorageServicePath+fileServerFetchVserverDetailsPath, url.PathEscape(id))
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s", ErrFileServerNotFound, id)
		}
		return nil, err
	}
	var env fileServerEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("parse get file server envelope: %w", err)
	}
	if env.Status != "" && env.Status != "success" {
		msg := strings.TrimSpace(env.Message)
		if msg == "" || strings.EqualFold(msg, env.Status) {
			var dataMsg string
			err := json.Unmarshal(env.Data, &dataMsg)
			if err != nil {
				return nil, fmt.Errorf("parse get file server message: %w", err)
			}
			if strings.TrimSpace(dataMsg) != "" {
				msg = strings.TrimSpace(dataMsg)
			}
		}
		if msg == "" {
			msg = env.Status
		}
		return nil, fmt.Errorf("get file server failed: %s (code: %d)", msg, env.ResponseCode)
	}
	var detail FileServerDetail
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &detail); err != nil {
			return nil, fmt.Errorf("parse get file server data: %w", err)
		}
		normalizeFileServerDetail(&detail)
		return &detail, nil
	}
	if err := json.Unmarshal(respBody, &detail); err != nil {
		return nil, fmt.Errorf("parse get file server body: %w", err)
	}
	normalizeFileServerDetail(&detail)
	return &detail, nil
}
