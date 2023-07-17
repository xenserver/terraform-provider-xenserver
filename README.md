![Continuous Integration](https://github.com/terra-farm/terraform-provider-xenserver/workflows/Continuous%20Integration/badge.svg)
[![Github Release](https://img.shields.io/github/release/ringods/terraform-provider-xenserver.svg)](link=https://github.com/terra-farm/terraform-provider-xenserver/releases)

# Terraform Provider for XenServer

## How to Build and Use

1. To compile the provider, clone this repository and run the following command from the repository root:
    ```bash
    go build
    ```
    This should create a binary called `terraform-provider-xenserver`.

2. Create the following directory in the user directory:
    ```bash
    .terraform.d/plugins/terraform.local/xenserver/0.3.1/<OS_ARCH>/
    ```
    The OS architecture will look something like `linux_amd64`. See [Terraform documentation](https://developer.hashicorp.com/terraform/registry/providers/os-arch) for recommended combinations.

3. Move the `terraform-provider-xenserver` binary into the new directory and rename to `terraform-provider-xenserver_v0.3.1`.
    ```bash
    mv terraform-provider-xenserver \
    ~/.terraform.d/plugins/terraform.local/xenserver/0.3.1/<OS_ARCH>/terraform-provider-xenserver_v0.3.1
    ```

This should now be recognised as a local provider by Terraform and can be set as a required provider in the following format:
```terraform
required_providers {
  xenserver = {
    source  = "terraform.local/xenserver"
    version = "0.3.1"
  }
}
```

## History

This repository was forked from [terra-farm/terraform-provider-xenserver](https://github.com/terra-farm/terraform-provider-xenserver) on 2nd June 2023 after ownership was transferred to XenServer. Many thanks to the original creators in the [terra-farm](https://github.com/terra-farm) project and their community for working on this.

---

NOTE: This has been updated to work with Terraform v.0.12.x

Website: [Xenserver Provider](https://terra-farm.github.io/provider-xenserver/)

If you want to add documentation, your starting point is the [modules](modules) folder.
