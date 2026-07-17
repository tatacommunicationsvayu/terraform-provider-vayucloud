// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

// Package client provides the HTTP client and shared constants for the VayuCloud API.
package client

import "time"

// API endpoint constants
const (
	// AuthURL is the authentication endpoint URL (POST JSON: email, password).
	// AuthURL = "https://ipcloud.tatacommunications.com/uat-portalservice/userdetail/getAuthToken"
	AuthURL = "https://api-uat.tatacommunications.com/v1/vayu/userdetail/getAuthToken"

	// DefaultAuthTokenLifetime is used when the auth API does not return expires_in.
	DefaultAuthTokenLifetime = 10 * time.Minute

	// APIURL is the base URL for the API
	APIURL = "https://ipcloud.tatacommunications.com/"
	// APIURL = "https://api-uat.tatacommunications.com/v1/vayu/"

	// DefaultTimeout is the default HTTP client timeout in seconds
	DefaultTimeout = 60

	// AuditPollInterval is the interval between audit status checks
	AuditPollInterval = 20 * time.Second

	// AuditPollMaxAttempts is the maximum number of polling attempts
	AuditPollMaxAttempts = 90

	AuditStatusCompleted  = "COMPLETED"
	AuditStatusInProgress = "IN PROGRESS"
	AuditStatusFailed     = "FAILED"
)
