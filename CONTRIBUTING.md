# Contributing

This provider is generated with Upjet from the released Terraform provider
schema and docs.

## Local Checks

```bash
make submodules
make generate.init
make go.build check-examples
make check-diff
```

When updating the Terraform provider version, update both Makefile pins:

- `TERRAFORM_PROVIDER_VERSION`
- `TERRAFORM_NATIVE_PROVIDER_BINARY`

Then clear cached upstream docs before regenerating:

```bash
rm -rf .work/pydantic/logfire
```

That cache is the main footgun: a stale local copy can make `make check-diff`
pass locally while CI fails from a fresh checkout.

## Release

Tag `main` with `vX.Y.Z`, push the tag, and watch `Publish Provider Package`.
Verify the package with:

```bash
docker buildx imagetools inspect ghcr.io/pydantic/provider-upjet-logfire:vX.Y.Z
```
