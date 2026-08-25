#!/usr/bin/env sh
# Import an existing load balancer; ID is the LB CI Master ID.
# After import, set zone_id in resource.tf (platform read does not return it).
#
#   LB_CI_ID=404212 terraform import vayucloud_network_lb.lb_haproxy "$LB_CI_ID"
#   LB_CI_ID=404212 ENGAGEMENT_ID=12345 terraform import vayucloud_network_lb.lb_haproxy "${LB_CI_ID}:${ENGAGEMENT_ID}"

: "${LB_CI_ID:?set LB_CI_ID}"
if [ -n "${ENGAGEMENT_ID:-}" ]; then
  terraform import vayucloud_network_lb.lb_haproxy "${LB_CI_ID}:${ENGAGEMENT_ID}"
else
  terraform import vayucloud_network_lb.lb_haproxy "$LB_CI_ID"
fi
