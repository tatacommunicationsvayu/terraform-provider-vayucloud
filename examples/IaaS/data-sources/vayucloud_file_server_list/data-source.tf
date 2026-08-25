# Copyright (c) TATA Communications
# SPDX-License-Identifier: Apache-2.0
#
# Example: list NAS vservers for an engagement and endpoint (getNASVservers/{engagement_id}/{endpoint_id}).

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

variable "engagement_id" {
  type        = number
  description = "Engagement id for getNASVservers path."
}

variable "endpoint_id" {
  type        = number
  description = "Endpoint id for getNASVservers path."
}

data "vayucloud_file_server_list" "all" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
}

output "vserver_ids" {
  value = [for v in data.vayucloud_file_server_list.all.vservers : v.vserver_id]
}

output "raw_response" {
  value     = data.vayucloud_file_server_list.all.raw_response
  sensitive = true
}
