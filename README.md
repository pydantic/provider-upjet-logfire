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

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-upjet-logfire
spec:
  package: xpkg.crossplane.io/pydantic/provider-upjet-logfire:v0.2.0
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

Run tests:

```console
go test ./...
```

Build:

```console
go build ./...
```
