# Changelog

## v0.3.0 (unreleased)

SECURITY:

- Bundle `terraform-provider-logfire` v0.2.3. The previous package bundled v0.1.8,
  whose binary was built with Go 1.25.4 and accepts the compromised eMudhra
  intermediate certificates (CVE-2026-111943). The controller binary was already
  rebuilt with Go 1.27.3, but the bundled provider makes the API calls and
  verifies the TLS certificate. It now reports go1.27.3.
- Check the release asset SHA256 before generation and the embedded provider's Go
  version with `make verify-native-provider`.

BREAKING CHANGES:

- `Alert`: `channelIds` is replaced by `channelAssignments`. Each entry has
  `channelId`, an optional `channelIdRef` and `channelIdSelector`, and an optional
  `scheduleId`. Replace a manifest such as `channelIds: [id1, id2]` with
  `channelAssignments: [{channelId: id1}, {channelId: id2}]`. Existing `Alert`
  resources must be updated before the upgrade.
- `Alert`: add `environments` to scope the query to deployment environments.

FEATURES:

- `Channel` `config` gains `channelId`, `includeAgentPrompt`, `installId`, `region`,
  `routingKeySecretRef`, and `serviceId`. This supports PagerDuty and Slack
  integration channels. The sensitive `routing_key` is read from a Kubernetes
  secret through `routingKeySecretRef`.

DOCUMENTATION:

- Show `writeConnectionSecretToRef` in the generated token examples and document
  that the token is written under the `attribute.token` key and cannot be read
  again.
- Record the bundled Terraform provider version in the README.

BUILD:

- Generate the provider schema from the checksum-verified release asset through a
  local filesystem mirror. Generation no longer depends on the Terraform Registry
  listing the release.
- Derive `TERRAFORM_NATIVE_PROVIDER_BINARY` from `TERRAFORM_PROVIDER_VERSION`.
- Pin golangci-lint to 2.15.0 to match CI. Version 2.7.2 cannot read Go 1.27
  export data.
- Consolidate the channel `config` sensitive-path fix into one generator rewrite
  that covers every sensitive field in the block.
- Remove the Terraform `schedule_id` interpolation from the generated `Alert`
  example. The provider does not generate a `logfire_schedule` resource.

## v0.2.3

- Build the controller with Go 1.27.3.

## v0.2.2

- Bundle `terraform-provider-logfire` v0.1.8.

## v0.2.1

- Bundle `terraform-provider-logfire` v0.1.6.
