# VayuCloud network public IP association example
#
# Associates tenant public NAT with zone|virtualmachine|firewall|baremetal|loadbalancer targets.

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

variable "associate_resource_type" {
  type        = string
  description = "Associate target URI segment: zone, virtualmachine, firewall, baremetal, loadbalancer."
}

variable "associate_resource_id" {
  type        = number
  description = "Platform ID for associate_resource_type."
}

variable "private_ip" {
  type        = string
  description = "PrivateIPv4 wired to NAT at associate time."
}

variable "public_ip_pricing_model" {
  type        = string
  description = "daily | monthly | reserved_* (see provider docs)."
  default     = "monthly"
}

variable "retain_on_dissociate" {
  type        = bool
  description = "Whether to preserve the EIP when tearing down associations."
  default     = false
}

resource "vayucloud_network_public_ip" "example" {
  resource_type           = var.associate_resource_type
  resource_id             = var.associate_resource_id
  private_ip              = var.private_ip
  public_ip_pricing_model = var.public_ip_pricing_model
  retain_on_dissociate    = var.retain_on_dissociate
}

output "association" {
  value = {
    public_ip = vayucloud_network_public_ip.example.public_ip
    audit_id  = vayucloud_network_public_ip.example.audit_id
  }
}
