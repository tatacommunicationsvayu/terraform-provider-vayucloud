# VayuCloud Virtual Machine Block Storage Data Source Example
#
# Reads an attached volume given instance ID and volume ID.

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
  description = "Virtual machine instance ID."
}

variable "volume_id" {
  type        = number
  description = "Attached volume (disk) ID on that instance."
}

data "vayucloud_virtualmachine_blockstorage" "example" {
  instance_id = var.instance_id
  volume_id   = var.volume_id
}

output "block_volume_summary" {
  description = "Attached block volume metadata"
  value = {
    id           = data.vayucloud_virtualmachine_blockstorage.example.id
    name         = data.vayucloud_virtualmachine_blockstorage.example.name
    size         = data.vayucloud_virtualmachine_blockstorage.example.size
    iops         = data.vayucloud_virtualmachine_blockstorage.example.iops
    disk_type    = data.vayucloud_virtualmachine_blockstorage.example.disk_type
    created_date = data.vayucloud_virtualmachine_blockstorage.example.created_date
  }
}
