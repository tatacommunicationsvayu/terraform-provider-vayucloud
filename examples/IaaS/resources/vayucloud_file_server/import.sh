#!/usr/bin/env sh
# Import an existing NAS vserver (file server) into Terraform state.
#
# Import ID (choose one):
#   1) VSERVER_ID alone — same as vayucloud_file_server.id after create.
#   2) ENGAGEMENT_ID/VSERVER_ID — recommended: stores engagement_id in state so destroy
#      and outputs (e.g. file_server_engagement_id) work even if refresh APIs fail.
#
# After import, run terraform plan and align endpoint_id, vserver_name, file_storage_type with the remote resource.
#
# Examples:
#   terraform import vayucloud_file_server.example 55394
#   terraform import vayucloud_file_server.example 15770/55394

terraform import vayucloud_file_server.example VSERVER_ID
