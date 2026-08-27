# Plugin for Terraform Provider for XenServer® Developer Guide

This guide helps developers set up a local environment, build, test, and extend the XenServer Terraform Provider while following the project's conventions. It is built on the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) in Go and published to the Terraform Registry as `registry.terraform.io/xenserver/xenserver`.

## Table of Contents
- [Conventions every contributor MUST follow](#conventions-every-contributor-must-follow)
- [Repository layout](#repository-layout)
- [Install dependencies](#install-dependencies)
- [Prepare the XenServer Go SDK (`goSDK/`)](#prepare-the-xenserver-go-sdk-gosdk)
- [Build the provider](#build-the-provider)
- [Generate documentation](#generate-documentation)
- [Run the provider locally (dev overrides)](#run-the-provider-locally-dev-overrides)
- [Logging](#logging)
- [Testing](#testing)
- [Add a new resource, data source, or function](#add-a-new-resource-data-source-or-function)
- [Handling Terraform Plugin Framework types](#handling-terraform-plugin-framework-types)
- [Linting](#linting)
- [Contributing changes](#contributing-changes)
- [Common errors](#common-errors)

## Conventions every contributor MUST follow

These are hard rules. All contributors must comply with all of them when changing provider code:

1. **Go naming.** Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`. Struct fields are `PascalCase` with `snake_case` Terraform tags, e.g. ``NameLabel types.String `tfsdk:"name_label"` ``. Struct-tag casing is enforced by the `tagliatelle` linter.
2. **No parentheses around conditions.** Write `if value == ""`, not `if (value == "")`.
3. **No gratuitous comments.** Do not add code comments unless they are necessary for clarity or explicitly requested.
4. **Terraform `types.String` values:** do not check for null. Check `value.ValueString() == ""` instead.
5. **Do not null-check a returned value before refreshing a computed property.**
6. **Always wrap and surface errors.** Wrap errors before returning them (the `wrapcheck` linter is enabled) and report user-facing failures through `resp.Diagnostics.AddError` / `resp.Diagnostics.AddAttributeError`. Never silently swallow an error.
7. **No `init()` functions** (`gochecknoinits`). Register new resources/data sources/functions in `xenserver/provider.go`.
8. **Schema.** Every `resource` must expose an `id` attribute in its Schema. Give every attribute a `MarkdownDescription`.
9. **Generated docs are not hand-edited.** Change the source/examples/templates, then run `make generate` and commit the regenerated `docs/`.
10. **Release notes:** do not include an auto-generated "Full Changelog", and do not reference internal ticket IDs or links — reference GitHub issue IDs only.
11. **Never commit** secrets, credentials, `.env`, Terraform state, or the local `goSDK/` contents (all git-ignored).
12. **Before every commit:** run `make lint` and `make generate`, and run the relevant acceptance tests (`make testacc`) when you change behavior.

## Repository layout

| Path | Contents |
|------|----------|
| `xenserver/` | Provider, resources, data sources, helpers, and tests. |
| `examples/` | Provider/resource/data-source examples; also the source for generated docs. `vm-main` and `pool-main` are runnable end-to-end samples. |
| `docs/` | Terraform Registry documentation, generated from schema + examples + `templates/`. Do not hand-edit. |
| `templates/`, `tools/` | Doc-generation template and tooling. |
| `GNUmakefile` | Developer entry points (build, lint, generate, test, apply…). |

File-per-component naming inside `xenserver/`:

- `<name>_resource.go` + `<name>_resource_test.go`
- `<name>_data_source.go` + `<name>_data_source_test.go`
- `<name>_utils.go` — model/type definitions and shared helpers for that component
- `provider.go` — provider schema, configuration, and component registration

## Install dependencies

- [Go](https://go.dev/doc/install) — version per [`go.mod`](./go.mod) (currently `1.25.8`)
- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.8
- `make`
- [`golangci-lint`](https://golangci-lint.run/) (or run it via Docker, shown in [Linting](#linting))

## Prepare the XenServer Go SDK (`goSDK/`)

The module maps the XenServer API package with `replace xenapi => ./goSDK`, so a populated `goSDK/` directory is required **before any build/lint/test/doc command**.

- [Download](https://www.xenserver.com/downloads) the XenServer SDK, unzip it, and copy everything under `XenServer-SDK/XenServerGo/src/` into a `goSDK/` directory at the provider root.
- CI performs this automatically; the download/extract sequence is the canonical reference for the exact steps.
- `goSDK/` is git-ignored — never commit its contents.

## Build the provider

```bash
make provider   # go mod tidy + go install . -> $GOBIN/terraform-provider-xenserver
make build      # go build -v ./...
```

## Generate documentation

Docs are generated with [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) from the schema, `examples/`, and `templates/`:

```bash
make generate   # go generate ./...
make doc        # tfplugindocs generate --provider-name xenserver
```

Run this whenever names, descriptions, schema, or examples change, then commit the updated `docs/`. CI fails if `git diff` shows uncommitted documentation changes.

## Run the provider locally (dev overrides)

Terraform loads your local build through a `dev_overrides` block in `~/.terraformrc`:

```bash
make ~/.terraformrc   # generates the dev_overrides file pointing at your $GOBIN
```

Or create `~/.terraformrc` manually:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/xenserver/xenserver" = "<your $GOBIN path>"
  }
  direct {}
}
```

Then drive a sample stack (override `WORKDIR` to choose the example directory):

```bash
make apply_vm                              # examples/vm-main
make apply_pool                            # examples/pool-main
make show_state resource=xenserver_vm.vm
make import resource=xenserver_vm.vm id=<vm-uuid>
make remove resource=xenserver_vm.vm
make destroy_vm
```

## Logging

```bash
export TF_LOG_PROVIDER="DEBUG"
export TF_LOG_PATH=/tmp/terraform.log
```

See [Managing Log Output](https://developer.hashicorp.com/terraform/plugin/log/managing).

## Testing

Acceptance tests create **real** XenServer resources and may incur cost. Provide a local `.env` (git-ignored) with the target environment:

```bash
export XENSERVER_HOST=https://<host-ip>
export XENSERVER_USERNAME=<username>
export XENSERVER_PASSWORD=<password>
export XENSERVER_INSECURE=<insecure>
export XENSERVER_CA_CERT_PATH=<ca-cert-path>
export NFS_SERVER=<nfs-ip>
export NFS_SERVER_PATH=<nfs-path>
export SMB_SERVER_PATH=<smb-path>
export SMB_SERVER_USERNAME=<smb-username>
export SMB_SERVER_PASSWORD=<smb-password>
export SUPPORTER_HOST=<supporter-ip>
export SUPPORTER_USERNAME=<supporter-username>
export SUPPORTER_PASSWORD=<supporter-password>
export SUPPORTER_INSECURE=<supporter-insecure>
export SUPPORTER_CA_CERT_PATH=<supporter-ca-cert-path>
```

```bash
make testacc    # TF_ACC=1 go test ./xenserver/ (+ pool tests), 60m timeout
make testpool   # pool acceptance tests only
```

> CI runs build, lint, and documentation-drift checks; it does not run the acceptance/unit tests (the unit tests currently depend on an unmocked XenAPI). Run the tests locally and record the results in your PR.

## Add a new resource, data source, or function

Follow this recipe exactly:

1. Create `xenserver/<name>_<type>.go`, where `<type>` is `resource`, `data_source`, or `function`. Each `resource` Schema must include an `id` attribute, and every attribute needs a `MarkdownDescription`.
2. Put the model struct and shared helpers in `xenserver/<name>_utils.go`.
3. Register the constructor (`New<Name>Resource` / `New<Name>DataSource` / `New<Name>Function`) in the matching list in `xenserver/provider.go` (`Resources` / `DataSources` / `Functions`).
4. Add an acceptance test `xenserver/<name>_<type>_test.go`.
5. Add an example:
   - `resource` → `examples/resources/xenserver_<name>/resource.tf` (plus `import.sh` if the resource is importable)
   - `data_source` → `examples/data-sources/xenserver_<name>/data-source.tf`
   - `provider` → `examples/provider/provider.tf`
6. Run `make generate` and commit the regenerated `docs/`.
7. Run `make lint` and the relevant `make testacc`.

## Handling Terraform Plugin Framework types

Use the framework types (`types.String`, `types.List`, `types.Set`, `types.Object`) in your models — not Go-native types — so Null/Unknown states are preserved (any `.tf` variable can be Unknown during `ValidateConfig`/`ModifyPlan`). Convert to Go-native values for logic, then convert back.

- Shared conversion and refresh helpers live in the `xenserver/*_utils.go` files (for example `host_utils.go`, `vm_utils.go`, `uuid_utils.go`). Reuse and extend those rather than re-implementing conversions.
- Initialise nested objects as **typed nulls** (never a bare empty struct) to avoid a `Value Conversion Error`.

Reference: [Accessing values](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/accessing-values).

## Linting

[`.golangci.yml`](./.golangci.yml) (golangci-lint v2) enables a strict set including `errcheck`, `wrapcheck`, `gosec`, `govet`, `staticcheck`, `tagliatelle` (struct-tag casing), `gochecknoinits`, and `exhaustive`, with `goimports` as the formatter.

```bash
gofmt -w -l xenserver/*.go
golangci-lint run ./...

# or via Docker, without a local install:
sudo docker run -it -v "$(pwd)":/app -w /app golangci/golangci-lint \
  golangci-lint run --config=/app/.golangci.yml
```

## Contributing changes

1. Fork the public repository <https://github.com/xenserver/terraform-provider-xenserver> and clone it; keep your local branch rebased on the latest upstream `master`.
2. Implement your change with tests, examples, and regenerated docs; run `make lint`, `make generate`, and `make testacc`.
3. Open a pull request that includes your acceptance-test results. A maintainer will trigger the internal test suite.
4. Two approvals are required before a PR is merged.

See [CONTRIBUTING.md](./CONTRIBUTING.md) for the full contribution policy.

## Common errors

- **`build constraints exclude all Go files` / `cannot find package "xenapi"`** — `goSDK/` is empty. Populate it (see [Prepare the XenServer Go SDK](#prepare-the-xenserver-go-sdk-gosdk)).
- **`error obtaining VCS status: exit status 128`** — run `git config --global --add safe.directory <path to repo>`.
- **CI "Unexpected difference in directories after code generation"** — you didn't regenerate docs. Run `make generate` and commit the updated `docs/`.
