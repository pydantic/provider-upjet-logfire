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

## Install

For complete examples, see:
- `examples/install.yaml`
- `examples/cluster/providerconfig/providerconfig.yaml`
- `examples/cluster/smoke/project.yaml`
- `examples/cluster/smoke/writetoken.yaml`

Per-resource generated reference examples are under `examples-generated/` for all supported resources.

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-upjet-logfire
spec:
  package: xpkg.crossplane.io/pydantic/provider-upjet-logfire:v0.2.1
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

## Developing

Generate code:

```console
make generate
```

Run the repo checks that gate CI:

```console
go test ./...
make check-examples
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
