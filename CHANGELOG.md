# Changelog

## (unreleased)

BREAKING CHANGES:

- On Logfire instances with managed dashboards enabled, a `Dashboard` created
  by this provider is locked against edits in the Logfire UI and through MCP.
  Set `managedAllowEdits: true` to allow those edits. Existing dashboards stay
  editable.
- Late-initialization no longer copies optional fields that have no Logfire
  default into the spec. Removing such a field, such as a description, an alert
  `instancePolicy`, or a spending limit window, now clears it, as in Terraform.
  To import an object, copy its current optional values into the spec, or the
  first reconcile clears them. Values that earlier late-initialization wrote to
  a spec stay until you remove them. Before upgrading, add to the spec any
  optional value that exists only in Logfire and should stay, such as a spending
  limit set in the UI on a resource whose reconciliation is paused.

FEATURES:

- Bundle `terraform-provider-logfire` v0.3.1.
- The credentials accept an optional `organization`. On a self-hosted instance,
  one instance-admin API key then acts in that organization. Changing it does
  not move existing resources. Each Terraform run exchanges the key for a
  token, and Logfire limits exchanges per IP address, so a `ProviderConfig`
  that sets `organization` suits about a dozen resources.
- `Project` adds `organization`, which creates the project in another
  organization through an instance-admin key, and reports `organizationId`.
  Changing `organization` to another existing organization is refused, because
  Logfire would delete the project to move it. Late-initialization does not copy
  `organization` into the spec, so a rename of the organization does not block
  later updates of a `Project` that leaves `organization` unset.
- `Dashboard` marks dashboards as managed and adds `managedAllowEdits`.
- `Alert` adds `instancePolicy` for per-occurrence notifications.
- `Channel` adds `email` channels with `recipients`, and `baseUrl` for
  Opsgenie channels outside the US region.
- Add `Endpoint`, `SpendingLimits`, and `ProjectSpendingLimits` in the `gateway`
  group. They manage an AI Gateway endpoint and its providers, and the
  organization and project spending limits. `Endpoint.providers[].providerId`
  takes the UUID of a Gateway provider created outside Crossplane, because this
  provider does not generate a Gateway provider resource.

BUG FIXES:

- An empty success response is an error when the provider reads the response
  body. A list read treated it as an empty list, so the provider could lose
  track of every object in it.
- A create that the Logfire public API rate limit refuses is retried after the
  `Retry-After` delay when that delay fits the provider's retry budget.
- An `Alert` that names its channel with `channelIdRef` or `channelIdSelector`
  keeps reconciling after later edits. Crossplane's reference resolver saved only
  the fields it changed with a server-side apply, which dropped `projectId` or
  `channelAssignments`. The `Alert` then failed validation on every reconcile and
  could not be deleted. The provider now resolves references itself and saves
  them with a merge patch that leaves other fields alone. An `Alert` already in
  that state recovers on its next reconcile, unless its deletion was requested
  before that. In that case, set `spec.forProvider.projectId` to the project's
  ID so the deletion can finish.

## v0.3.0 (2026-10-21)

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
