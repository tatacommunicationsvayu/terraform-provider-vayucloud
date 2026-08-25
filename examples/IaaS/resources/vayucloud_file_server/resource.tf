# Copyright (c) TATA Communications
# SPDX-License-Identifier: Apache-2.0
#
# VayuCloud File Server (NAS vserver)
#
# API: POST .../nas/createNasVserver/{engagementId}
# Commands: create = terraform apply | delete = terraform destroy

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

variable "engagement_id" {
  type        = number
  description = "Engagement ID for the NAS vserver."
}

variable "endpoint_id" {
  type        = number
  description = "Endpoint ID for the NAS vserver."
}

variable "vserver_name" {
  type        = string
  description = "NAS vserver name."
}

variable "file_storage_type" {
  type        = string
  description = "File storage type (for example NAS-NFS)."
}

resource "vayucloud_file_server" "example" {
  engagement_id     = var.engagement_id
  endpoint_id       = var.endpoint_id
  vserver_name      = var.vserver_name
  file_storage_type = var.file_storage_type
}

output "id" {
  description = "Platform vserver id (vserverId) after create"
  value       = vayucloud_file_server.example.id
}

output "engagement_id" {
  description = "engagementId"
  value       = vayucloud_file_server.example.engagement_id
}

output "endpoint_id" {
  description = "endpointId"
  value       = vayucloud_file_server.example.endpoint_id
}

output "vserver_name" {
  description = "vserverName"
  value       = vayucloud_file_server.example.vserver_name
}

output "file_storage_type" {
  description = "fileStorageType"
  value       = vayucloud_file_server.example.file_storage_type
}

output "audit_id" {
  description = "Audit id from createNasVserver"
  value       = vayucloud_file_server.example.audit_id
}

output "status" {
  description = "Audit status from the last operation"
  value       = vayucloud_file_server.example.status
}

output "api_create_payload" {
  description = "Equivalent createNasVserver request body"
  value = {
    endpointId      = vayucloud_file_server.example.endpoint_id
    engagementId    = vayucloud_file_server.example.engagement_id
    vserverName     = vayucloud_file_server.example.vserver_name
    fileStorageType = vayucloud_file_server.example.file_storage_type
  }
}
