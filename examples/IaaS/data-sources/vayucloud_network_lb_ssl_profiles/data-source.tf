# VayuCloud LB SSL Profiles Data Source Example

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

variable "load_balancer_id" {
  type        = string
  description = "Load Balancer CI Master ID."
}

data "vayucloud_network_lb_ssl_profiles" "certs" {
  load_balancer_id = var.load_balancer_id
}

output "ssl_profiles" {
  description = "Uploaded certificate names — use as certificate_name on HTTPS virtual services"
  value       = data.vayucloud_network_lb_ssl_profiles.certs.ssl_profiles
}
