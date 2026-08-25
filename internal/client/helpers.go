// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// StringToInt64 converts a string to int64.
func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// parseBandwidthValue parses a bandwidth string (e.g., "2Mbps", "1Gbps") and returns the value in Mbps.
func parseBandwidthValue(bandwidth string) (int64, error) {
	bandwidth = strings.TrimSpace(bandwidth)
	bandwidth = strings.ToLower(bandwidth)

	var multiplier int64 = 1
	var numStr string

	if strings.HasSuffix(bandwidth, "gbps") {
		multiplier = 1000
		numStr = strings.TrimSuffix(bandwidth, "gbps")
	} else if strings.HasSuffix(bandwidth, "mbps") {
		multiplier = 1
		numStr = strings.TrimSuffix(bandwidth, "mbps")
	} else if strings.HasSuffix(bandwidth, "kbps") {
		// kbps is less than 1 Mbps, treat as 0 for comparison purposes
		multiplier = 0
		numStr = strings.TrimSuffix(bandwidth, "kbps")
	} else {
		// Try to parse as plain number (assume Mbps)
		numStr = bandwidth
	}

	numStr = strings.TrimSpace(numStr)
	value, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid bandwidth format: %s", bandwidth)
	}

	return value * multiplier, nil
}

func ParseMinimumCommitmentValue(minimumCommitment string) (int64, error) {
	minimumCommitment = strings.TrimSpace(minimumCommitment)
	minimumCommitment = strings.ToLower(minimumCommitment)

	var numStr string
	if strings.HasSuffix(minimumCommitment, "gb") {
		numStr = strings.TrimSuffix(minimumCommitment, "gb")
	} else if strings.HasSuffix(minimumCommitment, "mb") {
		numStr = strings.TrimSuffix(minimumCommitment, "mb")
	} else if strings.HasSuffix(minimumCommitment, "tb") {
		numStr = strings.TrimSuffix(minimumCommitment, "tb")
	} else {
		numStr = minimumCommitment
	}
	value, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid bandwidth format: %s", minimumCommitment)
	}

	return value, nil
}

// ValidateThroughputAndBandwidth validates that firewall throughput is >= bandwidth.
// When bandwidth exceeds throughput, a warning is logged and no error is returned.
func ValidateThroughputAndBandwidth(ctx context.Context, throughput, bandwidth string) error {
	throughputValue, err := parseBandwidthValue(throughput)
	if err != nil {
		return fmt.Errorf("invalid firewall throughput: %w", err)
	}

	bandwidthValue, err := parseBandwidthValue(bandwidth)
	if err != nil {
		return fmt.Errorf("invalid bandwidth: %w", err)
	}

	if throughputValue < bandwidthValue {
		tflog.Warn(ctx, "internet bandwidth of higher value than firewall throughput is not supported, please upgrade firewall throughput as well to enjoy selected higher internet bandwidth", map[string]any{
			"throughput": throughput,
			"bandwidth":  bandwidth,
		})
	}

	return nil
}

func ValidateMinimumCommitment(ctx context.Context, throughput, minimumCommitment string) error {
	throughputValue, err := parseBandwidthValue(throughput)
	if err != nil {
		return fmt.Errorf("invalid firewall throughput: %w", err)
	}

	minimumCommitmentValue, err := ParseMinimumCommitmentValue(minimumCommitment)
	if err != nil {
		return fmt.Errorf("invalid minimum commitment: %w", err)
	}
	equivalentBandwidth, err := getDataTransferBandwidthTiers(minimumCommitmentValue)
	tflog.Debug(ctx, "equivalentBandwidth", map[string]any{
		"equivalentBandwidth": equivalentBandwidth,
	})
	bandwidthValue, err := parseBandwidthValue(equivalentBandwidth)
	if err != nil {
		return fmt.Errorf("invalid bandwidth: %w", err)
	}
	tflog.Debug(ctx, "bandwidthValue", map[string]any{
		"bandwidthValue": bandwidthValue,
	})
	tflog.Debug(ctx, "throughputValue", map[string]any{
		"throughputValue": throughputValue,
	})
	if throughputValue < bandwidthValue {
		tflog.Warn(ctx, "Maximum burst bandwidth for the selected minimum commitment is higher than firewall throughput, We Recommend to upgrade firewall throughput as well to enjoy selected higher minimum commitment", map[string]any{
			"throughput":               throughput,
			"minimumCommitment":        minimumCommitment,
			"Maximum burst bandwidth":  equivalentBandwidth,
		})
	}

	return nil
}

func getDataTransferBandwidthTiers(monthlyDataTransferGB int64) (string, error) {
	if monthlyDataTransferGB >= 15001 {
		return "1000Mbps", nil
	} else if monthlyDataTransferGB >= 10001 {
		return "1000Mbps", nil
	} else if monthlyDataTransferGB >= 5001 {
		return "750Mbps", nil
	} else if monthlyDataTransferGB >= 1001 {
		return "500Mbps", nil
	} else {
		return "300Mbps", nil
	}
}

func GetDataTransferBaseBandwidthTiers(monthlyDataTransferGB int64) ([]string, error) {
	if monthlyDataTransferGB >= 15001 {
		return []string{"300Mbps", "1000Mbps"}, nil
	} else if monthlyDataTransferGB >= 10001 {
		return []string{"125Mbps", "1000Mbps"}, nil
	} else if monthlyDataTransferGB >= 5001 {
		return []string{"75Mbps", "750Mbps"}, nil
	} else if monthlyDataTransferGB >= 1001 {
		return []string{"50Mbps", "500Mbps"}, nil
	} else {
		return []string{"30Mbps", "300Mbps"}, nil
	}
}
