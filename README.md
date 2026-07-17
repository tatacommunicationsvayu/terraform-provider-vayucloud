# Terraform Provider for VayuCloud

This is a Terraform provider for managing resources on the VayuCloud platform.

## Prerequisites

Install the following on the machine where you will **build** the provider and run **Terraform**:

| Requirement | Notes |
|-------------|--------|
| **[Go](https://go.dev/dl/)** | Match or exceed the version in [`go.mod`](go.mod) (currently Go 1.25+). Required to compile the provider from this repository. |
| **[Terraform CLI](https://developer.hashicorp.com/terraform/install)** | Version **1.0 or later** (CLI-only; you do not need Terraform Cloud). Used to plan and apply configurations that use this provider. |
| **Git** | Needed to clone this repository. |
| **Network** | Outbound HTTPS to the VayuCloud / portal APIs (and Terraform registry if you are not using dev overrides for the provider). |

**Operating system:** Builds are supported on Windows, Linux, and macOS. Use the appropriate binary name when building (see below).

**Shell / PATH:** Ensure `go` and `terraform` are on your `PATH`. On Windows, use PowerShell or Command Prompt; Terraform reads `%APPDATA%\terraform.rc` for CLI configuration.

## Building the provider

1. Clone the repository and open a terminal in the repository root.
2. Build the provider binary:

```shell
go build -o terraform-provider-vayucloud
```

On **Windows**, the same command produces `terraform-provider-vayucloud.exe` in the current directory. Terraform expects the plugin filename to match the pattern `terraform-provider-<name><optional .exe>`.

3. Confirm the binary exists in the project root (or note the folder where you placed it).

## Installing the provider locally

### Method 1: Development overrides (recommended)

Development overrides tell Terraform to load this provider from your disk **instead of** downloading it from the registry. This is the usual workflow when you are changing provider code and testing with real `.tf` files.

#### 1. Choose the override directory

Set the override value to the **absolute path of the directory** that contains the built provider executable (the repository root if you ran `go build` there).

Examples:

- Linux/macOS: `/home/you/projects/terraform-provider-vayucloud`
- Windows: `C:/Dev Projects/terraform-provider-vayucloud` (forward slashes are fine in `terraform.rc`)

#### 2. Create or edit the Terraform CLI configuration

Terraform reads **one** of these files (first match wins in practice; most people use the platform default):

| OS | Typical path |
|----|----------------|
| Linux / macOS | `~/.terraformrc` |
| Windows | `%APPDATA%\terraform.rc` (e.g. `C:\Users\<You>\AppData\Roaming\terraform.rc`) |

You can also point Terraform at a specific file with the environment variable **`TF_CLI_CONFIG_FILE`** (useful in CI or when you do not want a global config).

#### 3. Add the `dev_overrides` block

Use the **registry source** for this provider and your **absolute** path to the directory containing the binary:

```hcl
provider_installation {
  dev_overrides {
    "tatacommunicationsvayu/vayucloud" = "C:/path/to/terraform-provider-vayucloud"
  }

  # Allow Terraform to install other providers from the registry as usual
  direct {}
}
```

Replace the path with your real project directory. On Linux/macOS, use a path like `/home/you/terraform-provider-vayucloud`.

#### 4. Use matching `required_providers` in your Terraform configuration

Your root module should declare the same source string so Terraform selects the correct provider:

```hcl
terraform {
  required_providers {
    vayucloud = {
      source = "tatacommunicationsvayu/vayucloud"
    }
  }
}
```

With dev overrides, Terraform **does not** verify the provider version against the registry the same way as a release install. You may see warnings about version constraints; locally you can relax or remove strict `version` pins while testing.

#### 5. Run Terraform from your configuration directory

```shell
cd /path/to/your/terraform/config
terraform init
terraform plan
```

**Tips:**

- If Terraform still tries to download this provider, check for typos in the source (`tatacommunicationsvayu/vayucloud`), that the override path is a **directory** containing the built binary, and that you are using the same Terraform CLI that reads your `terraform.rc`.
- After changing provider code, rebuild (`go build -o terraform-provider-vayucloud`) and re-run `terraform plan` / `apply` (no need to `init` again unless you change backend or modules).

### Method 2: Local filesystem mirror (manual install)

If you prefer not to use `dev_overrides`, you can copy the built binary into Terraform’s [implied local mirror layout](https://developer.hashicorp.com/terraform/cli/config/config-file#filesystem_mirror) under your user plugin directory. You must match registry hostname, namespace, provider name, version, and OS/arch folders. This is more work for day-to-day provider development than Method 1.

Example layout (Linux, adjust for Windows plugin dir and version):

```shell
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/tatacommunicationsvayu/vayucloud/1.0.0/linux_amd64/
cp terraform-provider-vayucloud ~/.terraform.d/plugins/registry.terraform.io/tatacommunicationsvayu/vayucloud/1.0.0/linux_amd64/
```

## Using the provider

The provider block supports `username`, `password`, and optional `timeout` (HTTP timeout in seconds, default `60`). API endpoints are fixed in the provider; you do not set `auth_url` or `api_url` in Terraform. TLS certificate validation is always enabled (TLS 1.2+).

The example below looks up a **virtual machine image** and **flavor** for a **zone ID you supply** (for example via `terraform.tfvars`, `-var 'zone_id=...'`, or CI variables), then creates a **virtual machine** using those IDs. Replace the filter `values` with names that exist in your account.

```hcl
terraform {
  required_providers {
    vayucloud = {
      source  = "tatacommunicationsvayu/vayucloud"
      version = "~> 1.0"
    }
  }
}

provider "vayucloud" {
  username = "your-user@example.com"
  password = "your-password"
  # timeout  = 60
}

# Set at apply time, e.g. terraform apply -var="zone_id=20504"
# or use a terraform.tfvars file. You can also wire this from another
# data source (e.g. vayucloud_network_zone_list) instead of a variable.
variable "zone_id" {
  description = "Network zone ID where the VM will be created."
  type        = number
}

data "vayucloud_virtualmachine_image" "selected" {
  zone_id = var.zone_id

  filter {
    name   = "name"
    values = ["Your image display name from the catalog"]
  }
}

data "vayucloud_virtualmachine_flavor" "selected" {
  zone_id = var.zone_id

  filter {
    name   = "name"
    values = ["Your flavor SKU, e.g. A16"]
  }
  filter {
    name   = "os_model"
    values = ["Rocky Linux"]
  }
}

resource "vayucloud_virtualmachine" "example" {
  name       = "example-vm"
  vm_purpose = "WEB"

  image_id  = data.vayucloud_virtualmachine_image.selected.images[0].id
  flavor_id = data.vayucloud_virtualmachine_flavor.selected.flavors[0].id
  zone_id   = var.zone_id
  iops      = 1

  is_kdump_or_page_enabled = "No"
  usage_type               = "ppu"
  pricing_model            = "hourly"
}
```

See the [`examples`](examples/) directory for full configurations (disk layouts, zones, block storage, and more).

## Authentication

The provider authenticates against the TATA Communications IDP using the portal token API (implemented as a JSON POST that sends `email` and `password` and receives an access token):

1. POST to the configured `getAuthToken` endpoint with JSON body `{ "email", "password" }` (the Terraform `username` is sent as `email`).
2. The response includes an access token; the client uses it as a Bearer token for subsequent requests.
3. The client re-authenticates when the token is no longer valid.

Endpoint URLs and related defaults are defined in [`internal/provider/IaaS/common/constants.go`](internal/provider/IaaS/common/constants.go) (for example, the VayuCloud portal host used by this release).

### Environment variables

Configuration can be supplied via environment variables (Terraform attributes override these when set):

- `VAYU_USERNAME` — Username (maps to the `email` field in the token request)
- `VAYU_PASSWORD` — Password
- `VAYU_TIMEOUT` — HTTP client timeout in seconds (must be a positive integer)

## Development

### Running tests

```shell
go test ./...
```

### Generating documentation

```shell
go generate ./...
```
