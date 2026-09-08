# Terraform Provider XenServer
This repository is the [Terraform Provider of XenServer](https://registry.terraform.io/providers/xenserver/xenserver/latest/docs), using the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework), containing:

- `docs/`      The generated documentation.
- `examples/`  The examples of provider, resources and data sources.
- `tools/`     The tool files, like generate document tool.
- `xenserver/` The provider, resources, data sources and tests.
- Miscellaneous meta files.

## Using the Provider

See the [provider documentation on the Terraform Registry](https://registry.terraform.io/providers/xenserver/xenserver/latest/docs) and the [`examples/`](examples/) directory.

## Developing the Provider

Requirements:

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.8
- [Go](https://golang.org/doc/install) (see the version in [`go.mod`](go.mod))

The full developer guide — environment setup (including the XenServer Go SDK), build, run, debugging, testing, and the **conventions every contributor must follow** — is in **[DEVELOPER.md](DEVELOPER.md)**.

Quick start:

```shell
make provider   # build and install the provider binary
make doc        # regenerate the documentation
make testacc    # run acceptance tests (requires a XenServer instance and a local .env)
```

## Known Issue
1. If you are using Terraform provider v0.1.1, you might encounter compatibility issues after applying the XenServer 8 updates released to Early Access on 25 September 2024 and Normal on 2 October 2024. Terraform v0.1.2 resolves these compatibility issues.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Third Party Notice

See [THIRD_PARTY_NOTICE.md](THIRD_PARTY_NOTICE.md).

## License

The source code in this repository is licensed under the Mozilla Public License 2.0 (MPL-2.0). See [LICENSE.md](LICENSE.md).

<sub>Copyright © 2026. Citrix Systems, Inc.</sub>
