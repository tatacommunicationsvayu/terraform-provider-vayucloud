#!/usr/bin/env sh
# Import an existing network firewall rule into Terraform state.
# Import ID format: firewall_id,rule_id
#
# Usage:
#   FIREWALL_ID=372123 RULE_ID=45678 terraform import vayucloud_network_firewall_rule.ill_to_vcs "${FIREWALL_ID},${RULE_ID}"
#
# Or export once:
#   export FIREWALL_ID=372123
#   export RULE_ID=45678

: "${FIREWALL_ID:?set FIREWALL_ID to the firewall id}"
: "${RULE_ID:?set RULE_ID to the firewall rule id}"
terraform import vayucloud_network_firewall_rule.ill_to_vcs "${FIREWALL_ID},${RULE_ID}"
