#!/usr/bin/env sh
# Import an existing VM; ID is platform instance_id.
#
#   INSTANCE_ID=98765 KEY=vm_with_disks terraform import "vayucloud_virtualmachine.this[\"$KEY\"]" "$INSTANCE_ID"

: "${INSTANCE_ID:?set INSTANCE_ID}"
: "${KEY:?set KEY (map key from var.virtual_machines)}"
terraform import "vayucloud_virtualmachine.this[\"$KEY\"]" "$INSTANCE_ID"
