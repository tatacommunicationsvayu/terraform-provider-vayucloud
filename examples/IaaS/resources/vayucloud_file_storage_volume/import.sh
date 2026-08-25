#!/usr/bin/env sh
# Import an existing file storage volume into Terraform state.
# ID format: file_server_id,volume_id (comma-separated; both are the platform string ids,
# typically numeric strings matching vayucloud_file_server.id and the volume id from the API).
#
# After import, run terraform plan; Terraform refreshes name and size_gb from the API.
#
# Example:
#   terraform import vayucloud_file_storage_volume.data_volume 9876543210,11223344

terraform import vayucloud_file_storage_volume.data_volume FILE_SERVER_ID,VOLUME_ID
