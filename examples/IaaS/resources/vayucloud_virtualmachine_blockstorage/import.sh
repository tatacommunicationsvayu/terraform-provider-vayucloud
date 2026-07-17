#!/usr/bin/env sh
# Import an existing attached volume. ID format: instance_id,volume_id (comma-separated integers).
#
#   INSTANCE_ID=372880 VOLUME_ID=456789 terraform import vayucloud_virtualmachine_blockstorage.attached_volume "${INSTANCE_ID},${VOLUME_ID}"

: "${INSTANCE_ID:?set INSTANCE_ID}"
: "${VOLUME_ID:?set VOLUME_ID}"
terraform import vayucloud_virtualmachine_blockstorage.attached_volume "${INSTANCE_ID},${VOLUME_ID}"
