#!/usr/bin/env sh
# Import an existing virtual service.
# Import ID format: {load_balancer_id}/{virtual_service_name}
#
#   LB_CI_ID=404212 VS_NAME=app-vs-https terraform import vayucloud_network_lb_virtualservice.app_vs "${LB_CI_ID}/${VS_NAME}"

: "${LB_CI_ID:?set LB_CI_ID}"
: "${VS_NAME:?set VS_NAME}"
terraform import vayucloud_network_lb_virtualservice.app_vs "${LB_CI_ID}/${VS_NAME}"
