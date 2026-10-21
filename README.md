# Provider Logfire

`provider-upjet-logfire` is a [Crossplane](https://crossplane.io/) provider for the
Logfire API, generated with [Upjet](https://github.com/crossplane/upjet).

The current provider surface is intentionally small:

- `Alert`
- `Channel`
- `Dashboard`
- `Project`
- `ReadToken`
- `WriteToken`

The package bundles `terraform-provider-logfire` v0.2.3. The embedded Terraform
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
