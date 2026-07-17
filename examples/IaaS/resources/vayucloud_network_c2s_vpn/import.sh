#!/usr/bin/env sh
# Import an existing C2S VPN into Terraform state.
# The import ID is the numeric firewall_id (same scope as the VPN on the platform).
#
# Usage:
#   FIREWALL_ID=12345 terraform import vayucloud_network_c2s_vpn.example "$FIREWALL_ID"
#
# Or export once:
#   export FIREWALL_ID=12345

: "${FIREWALL_ID:?set FIREWALL_ID to the firewall id (number as string ok)}"
terraform import vayucloud_network_c2s_vpn.example "$FIREWALL_ID"
