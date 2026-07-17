# VayuCloud Virtual Machine Data Source Example
#
# Reads one virtual machine by instance ID.

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

data "vayucloud_virtualmachine" "example" {
  instance_id = "xxxx"
}

output "virtual_machine_summary" {
  description = "Key virtual machine attributes"
  value = {
    name          = data.vayucloud_virtualmachine.example.name
    ip            = data.vayucloud_virtualmachine.example.ip
    hostname      = data.vayucloud_virtualmachine.example.hostname
    power_status  = data.vayucloud_virtualmachine.example.power_status
    pricing_model = data.vayucloud_virtualmachine.example.pricing_model
    volumes      = data.vayucloud_virtualmachine.example.volumes
    vcpu         = data.vayucloud_virtualmachine.example.vcpu
    vram         = data.vayucloud_virtualmachine.example.vram
  }
}
