# VayuCloud Network LB SSL Profile Resource Example
#
# Upload PEM certificate and private key before creating an HTTPS virtual service.
# Generate files locally (see CERTIFICATE.txt in this folder); do not commit keys.

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
  description = "Load Balancer CI Master ID (from vayucloud_network_lb.id)."
}

variable "certificate_name" {
  type        = string
  description = "Certificate upload name (UI label; .pem added on storage)."
  default     = "my-cert"
}

variable "certificate_path" {
  type        = string
  description = "Local path to PEM certificate file (.pem, .crt, or .cert)."
  default     = "cert.pem"
}

variable "private_key_path" {
  type        = string
  description = "Local path to PEM private key file."
  default     = "key.pem"
}

resource "vayucloud_network_lb_ssl_profile" "client_cert" {
  load_balancer_id = var.load_balancer_id
  certificate_name = var.certificate_name
  certificate      = file(var.certificate_path)
  private_key      = file(var.private_key_path)
}

output "ssl_profile" {
  value = {
    id         = vayucloud_network_lb_ssl_profile.client_cert.id
    audit_id   = vayucloud_network_lb_ssl_profile.client_cert.audit_id
    status     = vayucloud_network_lb_ssl_profile.client_cert.status
    valid_from = vayucloud_network_lb_ssl_profile.client_cert.valid_from
    valid_to   = vayucloud_network_lb_ssl_profile.client_cert.valid_to
  }
}
