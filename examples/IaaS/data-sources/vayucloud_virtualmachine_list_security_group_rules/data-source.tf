# VayuCloud VM List Security Group Rules Data Source Example
#
# Lists security groups and rules on a VM, grouped by port:
#   ports[].security_groups[].rules[]

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

variable "security_group_name_filter" {
  type        = list(string)
  description = "Optional security_group_name filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_virtualmachine_list_security_group_rules" "vm" {
  instance_id = var.instance_id

  dynamic "filter" {
    for_each = length(var.security_group_name_filter) > 0 ? [var.security_group_name_filter] : []
    content {
      name   = "security_group_name"
      values = filter.value
    }
  }
}

output "ports" {
  value = data.vayucloud_virtualmachine_list_security_group_rules.vm.ports
}

output "first_port_sg_names" {
  value = length(data.vayucloud_virtualmachine_list_security_group_rules.vm.ports) > 0 ? [
    for sg in data.vayucloud_virtualmachine_list_security_group_rules.vm.ports[0].security_groups : sg.name
  ] : []
}
