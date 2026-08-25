# Copyright IBM Corp. 2026



# Terraform Provider Configuration for VayuCloud
#
# This example shows how to configure the VayuCloud provider.
# The provider authenticates using a username and password against the TATA Communications IDP.

terraform {
  required_version = ">= 1.0"

  required_providers {
    vayucloud = {
      # During development, use dev_overrides in ~/.terraformrc or %APPDATA%\terraform.rc
      # For production, this would be: "registry.terraform.io/tatacommunicationsvayu/vayucloud"
      source  = "tatacommunicationsvayu/vayucloud"
      version = "1.0.0" # Version constraint required even with dev_overrides
    }
  }
}

# Provider configuration
#
# You can configure the provider using:
# 1. Inline configuration (shown below)
# 2. Environment variables (VAYU_USERNAME, VAYU_PASSWORD)
# 3. A mix of both (environment variables are used as defaults, but inline values take precedence)
#
# The provider authenticates against the TATA Communications IDP and automatically
# manages Bearer token refresh for all API operations.
provider "vayucloud" {
  # Credentials - can be provided inline or via environment variables
  username = var.vayucloud_username
  password = var.vayucloud_password
}

# Variables for credentials
variable "vayucloud_username" {
  type        = string
  description = "VayuCloud username (email) for authentication. Can also be set via VAYU_USERNAME environment variable."
  sensitive   = true
}

variable "vayucloud_password" {
  type        = string
  description = "VayuCloud password for authentication. Can also be set via VAYU_PASSWORD environment variable."
  sensitive   = true
}
