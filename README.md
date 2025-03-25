# Terraform Provider XenServer
This repository is the [Terraform Provider of XenServer](https://registry.terraform.io/providers/xenserver/xenserver/latest/docs), using the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework), containing:

- `docs/`      The generated documentation.
- `examples/`  The examples of provider, resources and data sources.
- `tools/`     The tool files, like generate document tool.
- `xenserver/` The provider, resources, data sources and tests.
- Miscellaneous meta files.

## Developing the Provider
### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.8
- [Go](https://golang.org/doc/install) (see the go version in go.mod)


### Prepare Development Environment

The following instructions are based on Linux system.

- Prepare the local XenServer SDK module
    - [Download](https://www.xenserver.com/downloads) the XenServer SDK zip package and unzip
    - Create `goSDK/` directory under `terraform-provider-xenserver/`
    - Copy all source files under `XenServer-SDK/XenServerGo/src/` to `terraform-provider-xenserver/goSDK/` folder

### Build

Run the commands as follows:
```shell
go get -u all
go mod tidy
```

To compile the provider, run `"go install"`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

In short, you can just run `make provider` instead.

### Document

To generate or update documentation, run `make doc`.

### Log

Set up log with `github.com/hashicorp/terraform-plugin-log/tflog`. To enable logging during local developing run:

```shell
export TF_LOG_PROVIDER="DEBUG"
export TF_LOG_PATH=/tmp/terraform.log
```

Find more details in [Managing Log Output](https://developer.hashicorp.com/terraform/plugin/log/managing).

### Acceptance Test
In order to run the full suite of acceptance tests, prepare a local `.env` file like:

```shell
export XENSERVER_HOST=https://<xenserver-host-ip>
export XENSERVER_USERNAME=<username>
export XENSERVER_PASSWORD=<password>
export NFS_SERVER=<nfs-server-ip>
export NFS_SERVER_PATH=<nfs-server-path>
export SMB_SERVER_PATH=<smb-server-path>
export SMB_SERVER_USERNAME=<smb-server-username>
export SMB_SERVER_PASSWORD=<smb-server-password>
export SUPPORTER_HOST=<supporter-ip>
export SUPPORTER_USERNAME=<supporter-username>
export SUPPORTER_PASSWORD=<supporter-password>
```

Run `"make testacc"`. *Note:* Acceptance tests generate actual resources and frequently incur costs when run.

```shell
make testacc
```

## Prepare Terraform for local provider install

Terraform allows to use local provider builds by setting a `dev_overrides` block in a configuration file called `.terraformrc`. This block overrides all other configured installation methods.

1. Set `GOBIN` to the path where Go installs binaries or use the default path:

```shell
export GOBIN=$HOME/go/bin
go env GOBIN
```

2. Create a new file called `.terraformrc` in home directory (~). Change the <PATH> to the value returned from the go env `GOBIN` command above.

```shell
provider_installation {
  dev_overrides {
      "registry.terraform.io/xenserver/xenserver" = "<PATH>"
  }
  direct {}
}
```

3. To compile the provider, run `"go install ."`. This will build the provider and put the provider binary in the <GOBIN> directory.

4. Local test with terraform command, you'll first need Terraform installed on your machine (see [Requirements](#requirements) above). Go to `examples/terraform-main/` folder, update the `main.tf` with your own configuration, then run terraform commands like:

```shell
terraform plan
terraform apply -auto-approve

// show state
terraform state show xenserver_vm.vm

// remove state
terraform state rm xenserver_vm.vm

// import state with uuid
terraform import xenserver_vm.vm <xenserver_vm.vm.uuid>
terraform show

// change resource.tf data and re-apply
terraform apply -auto-approve

terraform destroy -auto-approve
```

5. Local Run Go lint check:

```shell
gofmt -w -l xenserver/*.go
sudo docker run -it -v $(pwd):/app -w /app golangci/golangci-lint bash
golangci-lint run --config=/app/.golangci.yml
```

## Known Issue
1. If you are using Terraform provider v0.1.1, you might encounter compatibility issues after applying the XenServer 8 updates released to Early Access on 25 September 2024 and Normal on 2 October 2024. Terraform v0.1.2 resolves these compatibility issues.

## Contributing

See [DEVELOP.md](DEVELOP.md)

## License

See [LICENSE.md](LICENSE.md)
