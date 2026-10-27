---
page_title: "Security Best Practices"
description: |-
  Security guidance for VayuCloud credentials, certificates, secrets, and Terraform/OpenTofu state.
---

# Security Best Practices

Treat provider configuration, state, plans, logs, and certificate material as sensitive operational data.

## Credential handling

Prefer `VAYU_USERNAME` and `VAYU_PASSWORD` environment variables or a CI secret store over literal credentials in `.tf` files. Provider arguments override environment variables when both are set.

Supply resource secrets (for example C2S VPN `vpn_user_initial_password`) through an approved secrets mechanism or `TF_VAR_*` environment variables. Do not commit `.tfvars` files containing credentials. If a local `.tfvars` file is used, ensure that it is excluded from source control, restrict its permissions, and rotate credentials if they are exposed.

## CI/CD

- Store credentials in the CI platform's encrypted secret store.
- Mask username, password, tokens, and API responses in job logs.
- Use short-lived or least-privilege identities where the platform supports them.
- Separate plan and apply permissions.
- Restrict who can approve applies affecting production engagements.

## Certificates and private keys

`vayucloud_network_lb_ssl_profile` reads PEM content with `file()` and uploads it to the load balancer. Keep certificate and private-key files outside the repository or inject them into the runner just before apply.

Never commit private keys, use them as ordinary output values, or print their contents during troubleshooting. Remove temporary copies after the run and rotate a key if it reaches source control, logs, plans, or shared state.

## State and plan protection

Terraform/OpenTofu state can contain passwords, tokens, assigned IPs, and other sensitive values even when the schema marks them sensitive. Protect state as if it were a credential:

- Use an encrypted remote backend with access control and versioning.
- Enable locking where supported.
- Restrict state readers and writers to the smallest required group.
- Encrypt state backups and remove obsolete copies.
- Do not upload `.terraform`, `terraform.tfstate`, `*.tfplan`, or debug logs to public artifacts.
- Review plan storage and CI retention policies.

Use a backend appropriate for your organization's security and recovery requirements. Local state is suitable only for controlled, temporary development.

## Troubleshooting safely

Use debug logging only for the shortest practical interval. Review logs before sharing them externally; authorization headers, request bodies, passwords, and platform response data may be sensitive.

When requesting support, provide the resource type, operation, audit identifier, timestamp, and sanitized diagnostic message. Do not send credentials, private keys, complete state files, or unredacted plans.
