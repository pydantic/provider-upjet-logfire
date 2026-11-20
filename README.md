# Provider Logfire

`provider-upjet-logfire` is a [Crossplane](https://crossplane.io/) provider for the
Logfire API, generated with [Upjet](https://github.com/crossplane/upjet).

The current provider surface is intentionally small:

- `Alert`
- `Channel`
- `Dashboard`
- `Endpoint`
- `Project`
- `ProjectSpendingLimits`
- `ReadToken`
- `SpendingLimits`
- `WriteToken`

The package bundles `terraform-provider-logfire` v0.3.0. The embedded Terraform
provider makes the API calls and verifies the Logfire TLS certificate, so its
release version and Go toolchain matter. `make verify-native-provider` checks the
minimum Go version.

## Install

Published provider packages are installed from GitHub Container Registry:

`ghcr.io/pydantic/provider-upjet-logfire`

For complete examples, see:

- `examples/install.yaml`
- `examples/cluster/providerconfig/providerconfig.yaml`
- `examples/cluster/smoke/project.yaml`
- `examples/cluster/smoke/writetoken.yaml`

Per-resource generated reference examples are under `examples-generated/` for all
supported resources.

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-upjet-logfire
spec:
  package: ghcr.io/pydantic/provider-upjet-logfire:v0.3.0
```

## Credentials

The provider expects credentials as JSON with:

- `api_key`
- optional `base_url`

Example secret payload:

```json
{
  "api_key": "pylf_v2_...",
  "base_url": "https://logfire.example.com"
}
```

Use an organization API key that is not bound to one project. A key bound to a
project can create other projects but cannot read them, so a new `Project` never
becomes ready. The key needs the scopes of the kinds you manage, as listed in the
`terraform-provider-logfire` documentation.

## Tokens

A read or write token value is returned only at creation. The provider keeps it
out of the resource status and writes it to the connection secret named by
`spec.writeConnectionSecretToRef`. The secret key is `attribute.token`.

```yaml
apiVersion: token.logfire.pydantic.dev/v1alpha1
kind: WriteToken
metadata:
  name: example
spec:
  writeConnectionSecretToRef:
    name: example-write-token
    namespace: crossplane-system
  forProvider:
    projectIdRef:
      name: example
```

If `writeConnectionSecretToRef` is omitted, or the secret is deleted later, the
token cannot be read again. Delete the resource and create it again to mint a new
token.

## Lifecycle

Resources follow Terraform's semantics, with these points to know:

- Removing an optional field from the spec clears it in Logfire. A token's
  `expiresAt` keeps its current value instead. A field with a Logfire default,
  such as `active` or `visibility`, keeps a non-default value but returns to the
  default from `false`, so set such a field rather than removing it.
- The first reconcile of an imported object clears any optional value the spec
  leaves out, such as a description or a spending limit window. To import without
  changing the object, create the resource with `managementPolicies: ["Observe"]`
  and the `crossplane.io/external-name` annotation set to its Terraform import ID,
  such as `<project_id>/<alert_id>` for an alert. The
  `terraform-provider-logfire` documentation lists the import ID of each
  resource. Once the resource is ready, the annotation holds the object's ID.
  Copy the values from `status.atProvider` into `spec.forProvider`, then remove
  `managementPolicies`. Read and write tokens cannot be imported.
- A `SpendingLimits` or `ProjectSpendingLimits` resource owns all three windows.
  A window it leaves out has no limit, and deleting the resource removes every
  limit.
- Changing a field that Terraform can only change by replacing the object is
  refused: a dashboard or endpoint `slug`, a `projectId`, or a token `expiresAt`.
  The resource reports `Synced=False` with a `prevent_destroy` error. Delete the
  resource and create it with the new value instead.

Delete a project's resources before the project. Logfire answers a deleted
project with the same 403 as a project the key cannot reach, so an alert,
dashboard, token, or `ProjectSpendingLimits` whose project is gone can be neither
observed nor deleted. A `ClusterUsage` keeps the project until the resources that
use it are gone:

```yaml
apiVersion: protection.crossplane.io/v1beta1
kind: ClusterUsage
metadata:
  name: example-alert-uses-project
spec:
  replayDeletion: true
  of:
    apiVersion: project.logfire.pydantic.dev/v1alpha1
    kind: Project
    resourceRef:
      name: example
  by:
    apiVersion: alert.logfire.pydantic.dev/v1alpha1
    kind: Alert
    resourceRef:
      name: example
```

If a resource is already stuck after its project was deleted, confirm the
project is gone and remove the resource's finalizer.

## Developing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the provider update workflow,
including generated docs and schema rules and the `.work/pydantic/logfire` cache
pitfall.

Generate code:

```console
make generate
```

Run the repo checks that gate CI:

```console
go test ./...
make check-examples
make verify-native-provider
make check-diff
```

Build and run the provider in a local Crossplane control plane:

```console
make local-deploy
```

Run end-to-end tests against a real Logfire account by providing the credentials
payload expected by `cluster/test/setup.sh`:

```console
UPTEST_EXAMPLE_LIST=examples/cluster/smoke/project.yaml \
UPTEST_CLOUD_CREDENTIALS='{"api_key":"pylf_v2_..."}' \
make uptest
```

The `End to End Testing` workflow needs the `UPTEST_CLOUD_CREDENTIALS` repository
secret. It uses an issue comment that starts with `/test-examples` and lists the
example paths.
