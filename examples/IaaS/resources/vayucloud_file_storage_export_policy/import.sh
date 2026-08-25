#!/usr/bin/env sh
# Import an existing NAS client export into Terraform state.
# ID format: file_storage_volume_id,volume_name,client_ip
#
# Example:
#   terraform import vayucloud_file_storage_export_policy.nas_client \
#     383464,testNasMount01,10.0.0.1

terraform import vayucloud_file_storage_export_policy.nas_client \
  383464,testNasMount01,10.0.0.1
