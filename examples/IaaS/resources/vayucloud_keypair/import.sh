#!/usr/bin/env sh
# Import an existing keypair; ID matches platform resource id exposed in APIs.
#
#   KEYPAIR_ID=123 terraform import vayucloud_keypair.created_keypair "$KEYPAIR_ID"

: "${KEYPAIR_ID:?set KEYPAIR_ID}"
terraform import vayucloud_keypair.created_keypair "$KEYPAIR_ID"
