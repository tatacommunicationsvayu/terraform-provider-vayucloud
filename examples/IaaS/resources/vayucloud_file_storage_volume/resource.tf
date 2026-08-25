# Copyright (c) TATA Communications
# SPDX-License-Identifier: Apache-2.0
#
# VayuCloud File Storage Volume
#
# API: POST .../nas/createNASVolume/{engagementId}
# Commands:
#   create = terraform apply
#   resize = change size_gb, then terraform apply
#   delete = terraform destroy

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
  description = "Engagement ID."
}

variable "endpoint_id" {
  type        = number
  description = "Endpoint ID."
}

variable "file_server_id" {
  type        = string
  description = "NAS vserver ID (file_server.id)."
}

variable "name" {
  type        = string
  description = "Volume name."
}

variable "size_gb" {
  type        = number
  description = "Volume size in GB."
}

variable "volume_type" {
  type        = string
  description = "Volume type (for example Flex Volume)."
}

variable "file_storage_type" {
  type        = string
  description = "File storage type (for example NFS)."
}

variable "usage_type" {
  type        = string
  description = "Usage type (for example reserved)."
}

variable "pricing_model" {
  type        = string
  description = "Pricing model (for example reserved_1)."
}

variable "iops" {
  type        = number
  description = "IOPS tier."
}

resource "vayucloud_file_storage_volume" "data_volume" {
  engagement_id     = var.engagement_id
  endpoint_id       = var.endpoint_id
  file_server_id    = var.file_server_id
  name              = var.name
  size_gb           = var.size_gb
  volume_type       = var.volume_type
  file_storage_type = var.file_storage_type
  usage_type        = var.usage_type
  pricing_model     = var.pricing_model
  iops              = var.iops
}

output "id" {
  description = "Platform volume CI id (volCi) after create"
  value       = vayucloud_file_storage_volume.data_volume.id
}

output "engagement_id" {
  description = "engagementId"
  value       = vayucloud_file_storage_volume.data_volume.engagement_id
}

output "endpoint_id" {
  description = "endpointId"
  value       = vayucloud_file_storage_volume.data_volume.endpoint_id
}

output "file_server_id" {
  description = "vserverId"
  value       = vayucloud_file_storage_volume.data_volume.file_server_id
}

output "name" {
  description = "volumeName"
  value       = vayucloud_file_storage_volume.data_volume.name
}

output "size_gb" {
  description = "volumeSize (GB)"
  value       = vayucloud_file_storage_volume.data_volume.size_gb
}

output "volume_type" {
  description = "volumeType"
  value       = vayucloud_file_storage_volume.data_volume.volume_type
}

output "file_storage_type" {
  description = "fileStorageType"
  value       = vayucloud_file_storage_volume.data_volume.file_storage_type
}

output "usage_type" {
  description = "usageType"
  value       = vayucloud_file_storage_volume.data_volume.usage_type
}

output "pricing_model" {
  description = "pricingModel"
  value       = vayucloud_file_storage_volume.data_volume.pricing_model
}

output "iops" {
  description = "iops"
  value       = vayucloud_file_storage_volume.data_volume.iops
}

output "audit_id" {
  description = "Audit id from createNASVolume / resizeNasVolume"
  value       = vayucloud_file_storage_volume.data_volume.audit_id
}

output "status" {
  description = "Audit status from the last operation"
  value       = vayucloud_file_storage_volume.data_volume.status
}

output "api_create_payload" {
  description = "Equivalent createNASVolume request body"
  value = {
    endpointId      = vayucloud_file_storage_volume.data_volume.endpoint_id
    engagementId    = vayucloud_file_storage_volume.data_volume.engagement_id
    volumeName      = vayucloud_file_storage_volume.data_volume.name
    volumeSize      = vayucloud_file_storage_volume.data_volume.size_gb
    volumeType      = vayucloud_file_storage_volume.data_volume.volume_type
    vserverId       = vayucloud_file_storage_volume.data_volume.file_server_id
    fileStorageType = vayucloud_file_storage_volume.data_volume.file_storage_type
    usageType       = vayucloud_file_storage_volume.data_volume.usage_type
    pricingModel    = vayucloud_file_storage_volume.data_volume.pricing_model
    iops            = vayucloud_file_storage_volume.data_volume.iops
  }
}
