# Contributing

This provider is generated with Upjet from the released Terraform provider
schema and docs.

## Local Checks

```console
make submodules
make generate.init
make go.build check-examples verify-native-provider
make check-diff
```

`make generate.init` downloads the pinned Terraform provider release asset, checks
its published SHA256 checksum, and serves it from a local filesystem mirror. The
recorded `config/schema.json` therefore describes the exact binary the image
bundles. Generation does not wait for the Terraform Registry to list a release.

When updating the Terraform provider version, change one variable:

- `TERRAFORM_PROVIDER_VERSION`

`TERRAFORM_NATIVE_PROVIDER_BINARY` is derived from it. Then clear cached upstream
docs before regenerating:

```console
rm -rf .work/pydantic/logfire
```

That cache is the main footgun. A stale local copy can make `make check-diff` pass
locally while CI fails from a fresh checkout.

## Embedded Provider Toolchain

The image bundles a prebuilt `terraform-provider-logfire` binary, so this
repository's Go version does not apply to it. `make verify-native-provider` reads
the binary's build metadata and fails when its Go version is older than
`TERRAFORM_PROVIDER_MIN_GO`. Raise that minimum only with the reasons recorded in
the Makefile comment.

## Release

Tag `main` with `vX.Y.Z`, push the tag, and watch `Publish Provider Package`.
Verify the package and both Go toolchains:

```console
VERSION=vX.Y.Z
IMAGE=ghcr.io/pydantic/provider-upjet-logfire:$VERSION
TF_VERSION=$(sed -n 's/^export TERRAFORM_PROVIDER_VERSION ?= //p' Makefile)

docker buildx imagetools inspect "$IMAGE"
docker pull "$IMAGE"
PLATFORM=$(docker image inspect --format '{{.Os}}_{{.Architecture}}' "$IMAGE")
docker create --name provider-check "$IMAGE"
docker cp provider-check:/usr/local/bin/provider ./provider
docker cp "provider-check:/terraform/provider-mirror/registry.terraform.io/pydantic/logfire/$TF_VERSION/$PLATFORM/terraform-provider-logfire_v$TF_VERSION" ./terraform-provider
docker rm -f provider-check

go version -m ./provider
go version -m ./terraform-provider
```
