---
page_title: "Appendices"
description: |-
  VayuCloud provider terminology, naming conventions, limitations, security notes, change history, and support guidance.
---

# Appendices

## Terminology and abbreviations

- **VayuCloud** — the cloud platform managed by this provider.
- **Virtual machine (VM)** — a compute instance placed in a network zone.
- **Asynchronous operation** — an API operation completed through an audit workflow; the provider polls until completion.
- **C2S VPN** — client-to-site VPN.
- **VCS** — the VayuCloud S3-compatible object-storage service.
- **VIP** — virtual IP address used by a load-balancer virtual service.
- **IOPS** — storage performance tier expressed as **1**, **3**, or **5** IOPS/GB in Terraform configuration (not an absolute IOPS count such as a fixed baseline). The platform derives effective throughput from tier and volume size. Do not use legacy absolute-IOPS values in examples until that model is available in production.
- **Certificate** — PEM-encoded certificate material; use “certificate” consistently rather than “cert” in configuration documentation.

## Naming conventions

Use lowercase snake case for Terraform addresses and provider arguments. Use stable, descriptive names for Terraform blocks, and include the environment when the same service exists more than once.

Provider type names always use the `vayucloud_` prefix. List data sources use a `_list` suffix; collection attributes use plural names such as `virtual_machines`, `images`, or `flavors`.

## Known limitations

- Platform operations can take minutes and may require long CI job timeouts.
- Some read APIs do not return every create-time argument; imported resources may need those arguments retained in configuration.
- Storage expansion is generally grow-only.
- Load-balancer virtual-service VIPs should be retained before creating dependent public-IP associations.
- Secrets returned only during creation cannot be recovered from a later read; retain them in an approved secret system when needed.

## Security considerations

See [Security Best Practices](security.md) for credential handling, certificate files, state protection, and safe troubleshooting.

## Change history

This documentation set is maintained with the provider source. Record user-visible changes here when resource behavior, supported attributes, examples, or security guidance changes.

## Support and escalation path

1. Validate the configuration and review the resource-specific behavior notes.
2. Check the audit identifier and sanitized error diagnostic.
3. Reproduce with the smallest configuration that still fails.
4. Escalate through the VayuCloud support process with provider version, Terraform/OpenTofu version, resource type, operation, timestamp, audit ID, and redacted logs.

Never include credentials, private keys, complete state files, or unredacted plans in a support request.
