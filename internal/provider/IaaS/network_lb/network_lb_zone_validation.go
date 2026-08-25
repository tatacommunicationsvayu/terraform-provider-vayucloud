package network_lb

import (
	"context"
	"fmt"

	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_zone"
)

func validateLBZoneID(zoneID int64) error {
	if zoneID <= 0 {
		return fmt.Errorf("zone_id must be a positive integer")
	}
	return nil
}

func validateLBZoneForFirewall(c *client.Client, ctx context.Context, zoneID, firewallID int64) error {
	if err := validateLBZoneID(zoneID); err != nil {
		return err
	}
	return network_zone.ValidateNetworkZoneOnFirewall(c, ctx, zoneID, firewallID)
}

// ValidateLBZoneForFirewall ensures the zone exists and belongs to the given firewall.
// Exported for virtual service and other LB module resources.
func ValidateLBZoneForFirewall(c *client.Client, ctx context.Context, zoneID, firewallID int64) error {
	return validateLBZoneForFirewall(c, ctx, zoneID, firewallID)
}
