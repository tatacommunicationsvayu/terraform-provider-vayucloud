# VayuCloud Virtual Machine Security Group Association Example
#
# Attach one or more security groups to a VM.
# Create:  POST  .../security-group/vm/{instance_id}
# Destroy: DELETE .../security-group/vm/{instance_id}
# Both wait for audit completion without action-state.

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

variable "security_group_ids" {
  type        = list(string)
  description = "Security group IDs to attach to the VM."
}

resource "vayucloud_virtualmachine_security_group_association" "example" {
  instance_id        = var.instance_id
  security_group_ids = var.security_group_ids
}

output "association" {
  value = {
    id                 = vayucloud_virtualmachine_security_group_association.example.id
    instance_id        = vayucloud_virtualmachine_security_group_association.example.instance_id
    security_group_ids = vayucloud_virtualmachine_security_group_association.example.security_group_ids
    audit_id           = vayucloud_virtualmachine_security_group_association.example.audit_id
    status             = vayucloud_virtualmachine_security_group_association.example.status
  }
}
