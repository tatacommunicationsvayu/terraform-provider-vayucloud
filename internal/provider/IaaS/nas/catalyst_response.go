// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"encoding/json"
	"strings"
)

type catalystEnvelope struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
	Data         json.RawMessage `json:"data"`
}

// catalystFailureMessage extracts a user-facing error from CatalystResponse.
// Portal often puts the real text in data (e.g. validateNASOrder) while message is just "failed".
func catalystFailureMessage(env catalystEnvelope) string {
	if detail := catalystDataAsString(env.Data); detail != "" &&
		!strings.EqualFold(strings.TrimSpace(detail), strings.TrimSpace(env.Message)) {
		return detail
	}
	if msg := strings.TrimSpace(env.Message); msg != "" && !strings.EqualFold(msg, "failed") && !strings.EqualFold(msg, "success") {
		return msg
	}
	if detail := catalystDataAsString(env.Data); detail != "" {
		return detail
	}
	if msg := strings.TrimSpace(env.Message); msg != "" {
		return msg
	}
	if env.ResponseCode != 0 {
		return "request failed"
	}
	return "request failed"
}

func catalystDataAsString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(string(raw))
}

func parseCatalystEnvelope(raw []byte) (catalystEnvelope, error) {
	var env catalystEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return catalystEnvelope{}, err
	}
	return env, nil
}
