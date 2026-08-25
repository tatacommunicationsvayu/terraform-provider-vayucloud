package network_lb_ssl_profile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

type SSLProfileCreateRequest struct {
	LoadBalancerCI       int64
	CertificateName      string
	CertificateContent   string
	PrivateKeyContent    string
	PrivateKeyPassphrase string
}

func compositeSSLProfileID(lbCiID int64, certificateName string) string {
	return fmt.Sprintf("%d/%s", lbCiID, strings.TrimSpace(certificateName))
}

func parseCompositeSSLProfileID(id string) (int64, string, bool) {
	id = strings.TrimSpace(id)
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	lbCiID, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || lbCiID <= 0 {
		return 0, "", false
	}
	certName := strings.TrimSpace(parts[1])
	if certName == "" {
		return 0, "", false
	}
	return lbCiID, certName, true
}

// stripPEMSuffix returns the upload base name (backend create expects name without .pem).
func stripPEMSuffix(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasSuffix(strings.ToLower(name), ".pem") {
		return name[:len(name)-4]
	}
	return name
}

// ensurePEMSuffix returns the HAProxy list/delete storage name (e.g. test.pem).
func ensurePEMSuffix(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	if strings.HasSuffix(strings.ToLower(name), ".pem") {
		return name
	}
	return name + ".pem"
}

func parseLBServiceEnvelope(respBody []byte) (map[string]interface{}, error) {
	var envelope map[string]interface{}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	status, _ := envelope["status"].(string)
	if !strings.EqualFold(strings.TrimSpace(status), "success") {
		msg, _ := envelope["message"].(string)
		if msg == "" {
			msg = "request failed"
		}
		return nil, fmt.Errorf("status %s: %s", status, msg)
	}
	return envelope, nil
}

func extractLBServiceProfileValues(envelope map[string]interface{}) ([]map[string]interface{}, error) {
	data, err := extractLBServiceDataObject(envelope)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(stringFromActionStateMap(data, "status"), "error") {
		msg := stringFromActionStateMap(data, "response", "message")
		if msg == "" {
			msg = "profile list failed"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	raw, ok := data["values"]
	if !ok || raw == nil {
		return []map[string]interface{}{}, nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected values type %T", raw)
	}
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result, nil
}

func extractLBServiceDataObject(envelope map[string]interface{}) (map[string]interface{}, error) {
	raw, ok := envelope["data"]
	if !ok || raw == nil {
		return map[string]interface{}{}, nil
	}
	data, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data type %T", raw)
	}
	return data, nil
}

// ListClientSSLProfiles returns uploaded client SSL certificates on the load balancer.
func ListClientSSLProfiles(c *client.Client, ctx context.Context, lbCiMasterID int64) ([]map[string]interface{}, error) {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/list/client/sslprofile/%d", lbCiMasterID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	envelope, err := parseLBServiceEnvelope(respBody)
	if err != nil {
		return nil, err
	}
	return extractLBServiceProfileValues(envelope)
}

func postSSLProfileAudit(c *client.Client, ctx context.Context, method, path string, body io.Reader, contentType string) (*client.AuditResponse, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	url := client.APIURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Vayu-Client-Id", "vayu_iac")

	resp, err := c.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse SSL profile response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("ssl profile request failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	return &result, nil
}

func buildSSLProfileMultipartBody(req *SSLProfileCreateRequest) (io.Reader, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("certificateName", req.CertificateName); err != nil {
		return nil, "", err
	}
	if req.PrivateKeyPassphrase != "" {
		if err := writer.WriteField("privateKeyPassphrase", req.PrivateKeyPassphrase); err != nil {
			return nil, "", err
		}
	}

	if err := writeMultipartFile(writer, "certificate", req.CertificateName+".crt", req.CertificateContent); err != nil {
		return nil, "", err
	}
	if err := writeMultipartFile(writer, "privateKey", req.CertificateName+".key", req.PrivateKeyContent); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &body, writer.FormDataContentType(), nil
}

func writeMultipartFile(writer *multipart.Writer, fieldName, fileName, content string) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	header.Set("Content-Type", "application/octet-stream")
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = io.WriteString(part, content)
	return err
}

// CreateSSLProfile uploads an SSL client profile to HAProxy on the load balancer.
func CreateSSLProfile(c *client.Client, ctx context.Context, req *SSLProfileCreateRequest) (*client.AuditResponse, error) {
	tflog.Info(ctx, "Creating SSL Profile", map[string]any{
		"certificate_name": req.CertificateName,
		"load_balancer_id": req.LoadBalancerCI,
	})

	body, contentType, err := buildSSLProfileMultipartBody(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build multipart request: %w", err)
	}

	path := fmt.Sprintf(common.LoadBalancerServicePath+"/create/ssl/clientprofile/%d", req.LoadBalancerCI)
	result, err := postSSLProfileAudit(c, ctx, http.MethodPost, path, body, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to create ssl profile: %w", err)
	}

	tflog.Info(ctx, "SSL Profile creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
	})
	return result, nil
}

// DeleteSSLProfile removes an SSL profile by HAProxy storage name (must include .pem, as in list API).
func DeleteSSLProfile(c *client.Client, ctx context.Context, lbCiMasterID int64, storageName string) (*client.AuditResponse, error) {
	storageName = strings.TrimSpace(storageName)
	tflog.Info(ctx, "Deleting SSL Profile", map[string]any{
		"load_balancer_id": lbCiMasterID,
		"certificate_name": storageName,
	})

	path := fmt.Sprintf(
		common.LoadBalancerServicePath+"/delete/ssl/clientprofile/%d/%s",
		lbCiMasterID,
		url.PathEscape(storageName),
	)
	result, err := postSSLProfileAudit(c, ctx, http.MethodDelete, path, nil, "application/json")
	if err != nil {
		return nil, fmt.Errorf("failed to delete ssl profile: %w", err)
	}

	tflog.Info(ctx, "SSL Profile deletion initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
	})
	return result, nil
}

// GetSSLProfileDetails fetches SSL profile state from action-state API.
// Action-state uses the upload base name (without .pem).
func GetSSLProfileDetails(c *client.Client, ctx context.Context, lbCiMasterID int64, storageName string) (map[string]interface{}, error) {
	body := map[string]any{
		"resourceId":      lbCiMasterID,
		"certificateName": stripPEMSuffix(storageName),
	}

	resp, err := common.UpdateActionState(ctx, c, "sslprofile", "read", body)
	if err != nil {
		return nil, fmt.Errorf("failed to get ssl profile details: %w", err)
	}

	if resp.Status != "SUCCESS" && resp.Status != "success" {
		msg := resp.Message
		if msg == "" {
			msg = "no message"
		}
		return nil, fmt.Errorf("status %s: %s", resp.Status, msg)
	}

	var details map[string]interface{}
	if err := json.Unmarshal(resp.Data, &details); err != nil {
		return nil, fmt.Errorf("failed to parse ssl profile details: %w", err)
	}
	return details, nil
}

// FindSSLProfileOnLB finds a cert by exact storage name (e.g. test.pem).
func FindSSLProfileOnLB(c *client.Client, ctx context.Context, lbCiMasterID int64, storageName string) (map[string]interface{}, error) {
	profiles, err := ListClientSSLProfiles(c, ctx, lbCiMasterID)
	if err != nil {
		return nil, err
	}
	target := ensurePEMSuffix(storageName)
	for _, profile := range profiles {
		name := stringFromActionStateMap(profile, "name", "storage_name")
		if strings.EqualFold(name, target) {
			return profile, nil
		}
	}
	return nil, fmt.Errorf("ssl profile %q not found on load balancer %d", target, lbCiMasterID)
}

func stringFromActionStateMap(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch val := v.(type) {
		case string:
			if val != "" {
				return val
			}
		default:
			s := fmt.Sprint(val)
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}







