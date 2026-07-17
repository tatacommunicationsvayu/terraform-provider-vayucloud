#!/usr/bin/env sh
# Import an existing zone; ID is platform network_zone_id.
#
#   NETWORK_ZONE_ID=12345 terraform import vayucloud_network_zone.auto_ipam_zone "$NETWORK_ZONE_ID"

: "${NETWORK_ZONE_ID:?set NETWORK_ZONE_ID}"
terraform import vayucloud_network_zone.auto_ipam_zone "$NETWORK_ZONE_ID"
