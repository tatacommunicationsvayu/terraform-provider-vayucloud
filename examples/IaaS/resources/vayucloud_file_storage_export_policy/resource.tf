# Copyright (c) TATA Communications
# SPDX-License-Identifier: Apache-2.0
#
# VayuCloud File Storage Export Policy (NAS client attach / detach)
#
# API: POST .../nas/attachClient  (apply)
#      DELETE .../nas/volumes/{volumeId}/clients/{clientIp} (destroy)
#
# Required on the resource:
#   file_storage_volume_id  -> nasVolCi
#   name                    -> volume name
#   client_ip               -> client IP allowed to mount
#
# Drift detection (optional, recommended):
#   Set engagement_id + file_server_id (same as the parent volume).
#   On refresh/plan, if the portal detached client_ip, the provider drops the
#   resource from state and the next plan proposes recreate (re-attach).

terraform {
  required_providers {
    vayucloud = {
      source  = "tatacommunicationsvayu/vayucloud"
      version = "1.0.0"
    }
  }
}

provider "vayucloud" {
  username = var.vayucloud_username
  password = var.vayucloud_password
}

variable "vayucloud_username" {
  type        = string
  description = "VayuCloud API username (or set VAYU_USERNAME)."
  sensitive   = true
}

variable "vayucloud_password" {
  type        = string
  description = "VayuCloud API password (or set VAYU_PASSWORD)."
  sensitive   = true
}

variable "file_storage_volume_id" {
  type        = string
  description = "NAS volume CI id (nasVolCi)."
}

variable "file_storage_volume_name" {
  type        = string
  description = "NAS volume name."
}

variable "client_ip" {
  type        = string
  description = "Client IP allowed to mount the volume."
}

variable "engagement_id" {
  type        = number
  description = "Engagement ID for drift detection. Omit (null) when not needed."
  default     = null
  nullable    = true
}

variable "file_server_id" {
  type        = string
  description = "NAS vserver ID for drift detection. Omit (null) when not needed."
  default     = null
  nullable    = true
}

resource "vayucloud_file_storage_export_policy" "nas_client" {
  file_storage_volume_id = var.file_storage_volume_id
  name                   = var.file_storage_volume_name
  client_ip              = var.client_ip
  engagement_id          = var.engagement_id
  file_server_id         = var.file_server_id
}

output "id" {
  description = "Synthetic id {file_storage_volume_id}/{client_ip}"
  value       = vayucloud_file_storage_export_policy.nas_client.id
}

output "client_ip" {
  value = vayucloud_file_storage_export_policy.nas_client.client_ip
}

output "audit_id" {
  value = vayucloud_file_storage_export_policy.nas_client.audit_id
}
