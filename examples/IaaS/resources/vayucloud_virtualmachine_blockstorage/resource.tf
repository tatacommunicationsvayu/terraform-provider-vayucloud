# VayuCloud virtual machine block storage resource example
#
# Attaches a data volume to an existing virtual machine instance.

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
  type        = string
  description = "VayuCloud API username (or set VAYU_USERNAME)."
  sensitive   = true
}

variable "vayucloud_password" {
  type        = string
  description = "VayuCloud API password (or set VAYU_PASSWORD)."
  sensitive   = true
}

variable "instance_id" {
  type        = number
  description = "Target virtual machine instance_id."
}

variable "volume_display_name" {
  type        = string
  description = "Attached volume logical name visible in Terraform state."
}

variable "volume_size" {
  type        = number
  description = "Provisioned disk size units as required by API (adjust per provider docs)."
}

variable "volume_iops" {
  type        = number
  description = "IOPS tier for the volume."
  default     = 1
}

resource "vayucloud_virtualmachine_blockstorage" "attached_volume" {
  instance_id = var.instance_id
  name        = var.volume_display_name
  size        = var.volume_size
  iops        = var.volume_iops
}

output "attached_volume" {
  value = {
    id           = vayucloud_virtualmachine_blockstorage.attached_volume.id
    disk_type    = vayucloud_virtualmachine_blockstorage.attached_volume.disk_type
    created_date = vayucloud_virtualmachine_blockstorage.attached_volume.created_date
    audit_id     = vayucloud_virtualmachine_blockstorage.attached_volume.audit_id
    status       = vayucloud_virtualmachine_blockstorage.attached_volume.status
  }
}
