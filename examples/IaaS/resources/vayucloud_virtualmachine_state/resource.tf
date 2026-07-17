# VayuCloud virtual machine power state resource example
#
# Runs a supported power action on an existing VM instance.

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
  description = "Existing virtual machine instance_id."
}

variable "power_action" {
  type        = string
  description = "One of power_off, power_on, suspend, hard_reboot, soft_reboot, resume."
}

resource "vayucloud_virtualmachine_state" "example" {
  instance_id = var.instance_id
  action      = var.power_action
}

output "power" {
  value = {
    power_status = vayucloud_virtualmachine_state.example.power_status
    audit_id     = vayucloud_virtualmachine_state.example.audit_id
    status       = vayucloud_virtualmachine_state.example.status
  }
}
