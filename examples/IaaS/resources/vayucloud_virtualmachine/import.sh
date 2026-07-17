#!/usr/bin/env sh
# Import an existing VM; ID is platform instance_id.
#
#   INSTANCE_ID=98765 terraform import vayucloud_virtualmachine.vm_with_disks "$INSTANCE_ID"

: "${INSTANCE_ID:?set INSTANCE_ID}"
terraform import vayucloud_virtualmachine.vm_with_disks "$INSTANCE_ID"
