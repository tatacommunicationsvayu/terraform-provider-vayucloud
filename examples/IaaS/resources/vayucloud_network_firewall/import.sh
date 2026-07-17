#!/usr/bin/env sh
# Import an existing firewall; ID is platform network_firewall_id.
#
#   NETWORK_FIREWALL_ID=330765 terraform import vayucloud_network_firewall.terraform_firewall "$NETWORK_FIREWALL_ID"

: "${NETWORK_FIREWALL_ID:?set NETWORK_FIREWALL_ID}"
terraform import vayucloud_network_firewall.terraform_firewall "$NETWORK_FIREWALL_ID"
