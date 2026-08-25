#!/usr/bin/env sh
# Import an existing SSL profile on a load balancer.
# Import ID format: {load_balancer_id}/{certificate_name.pem}
#
#   LB_CI_ID=404212 CERT_NAME=my-cert.pem terraform import vayucloud_network_lb_ssl_profile.client_cert "${LB_CI_ID}/${CERT_NAME}"

: "${LB_CI_ID:?set LB_CI_ID}"
: "${CERT_NAME:?set CERT_NAME (e.g. my-cert.pem)}"
terraform import vayucloud_network_lb_ssl_profile.client_cert "${LB_CI_ID}/${CERT_NAME}"
