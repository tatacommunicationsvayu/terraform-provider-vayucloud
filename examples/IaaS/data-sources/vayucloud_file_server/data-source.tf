# Copyright (c) TATA Communications
# SPDX-License-Identifier: Apache-2.0
#
# Example: read an existing NAS vserver by platform id (same value as vayucloud_file_server.id).

terraform {
  required_providers {
    vayucloud = {
      source = "tatacommunicationsvayu/vayucloud"
    }
  }
}

provider "vayucloud" {
  username = var.vayucloud_username
  password = var.vayucloud_password
}

variable "vayucloud_username" {
  type = string
}

variable "vayucloud_password" {
  type      = string
  sensitive = true
}

variable "nas_vserver_id" {
  type        = string
  description = "Platform NAS vserver id (digits only, e.g. from portal or vayucloud_file_server.id)."
}

data "vayucloud_file_server" "example" {
  vserver_id = var.nas_vserver_id
}

output "file_server_name" {
  value = data.vayucloud_file_server.example.file_server_name
}

output "engagement_id" {
  value = data.vayucloud_file_server.example.engagement_id
}

output "endpoint_id" {
  value = data.vayucloud_file_server.example.endpoint_id
}

output "file_storage_type" {
  value = data.vayucloud_file_server.example.file_storage_type
}
