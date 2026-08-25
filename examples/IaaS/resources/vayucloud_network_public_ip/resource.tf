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

variable "resource_type" {
  type        = string
  description = "Target type (for example virtualmachine, firewall, loadbalancer, zone)."
}

variable "resource_id" {
  type        = number
  description = "Target resource ID."
}

variable "private_ip" {
  type        = string
  description = "Private IP to NAT."
}

variable "public_ip_pricing_model" {
  type        = string
  description = "Public IP pricing model (for example daily)."
}

variable "retain_on_dissociate" {
  type        = bool
  description = "Retain the public IP when the association is destroyed."
}

resource "vayucloud_network_public_ip" "example" {
  resource_type           = var.resource_type
  resource_id             = var.resource_id
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
