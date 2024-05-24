# Terraform Provider XenServer

This repository the terraform provider of XenServer, using the Terraform Plugin Framework(https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework), containing:

- `docs/`      The generated documentation.
- `examples/`  The examples of provider, resources and data sources.
- `tools/`     The tool files, like generate document tool.
- `xenserver/` The provider, resources and data sources.(include tests)
- Miscellaneous meta files.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22.2

## Developing the Provider

### Prepare

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To build the provider, you'll need to prepare the local XenServer module for Go. 
- Download the XenServer SDK zip package and unzip
- Create goSDK directory under terraform-provider-xenserver/
- Copy all source files under XenServer-SDK/XenServerGo/src/ to terraform-provider-xenserver/goSDK/ folder

### Build

Run the commands as follows:

```shell
go get -u all
go mod tidy
```

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

### Document

To generate or update documentation, run `go generate`.

### Log

Set up log with `github.com/hashicorp/terraform-plugin-log/tflog`. To enable logging during local develping run `export TF_LOG_PROVIDER="DEBUG"`. (See https://developer.hashicorp.com/terraform/plugin/log/managing.)

### Test
In order to run the full suite of Acceptance tests, prepare a local .env file like:

```shell
export XENSERVER_HOST="https://<xenserver-host-ip>"
export XENSERVER_USERNAME="<username>"
export XENSERVER_PASSWORD="<password>"
```

Run `make testacc`. 
*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

## Prepare Terraform for local provider install

Terraform allows to use local provider builds by setting a dev_overrides block in a configuration file called .terraformrc. This block overrides all other configured installation methods.

1. Set GOBIN to the path where Go installs binaries or use the default path:

```shell
export GOBIN=/Users/<Username>/go/bin
go env GOBIN
```

2. Create a new file called .terraformrc in home directory (~). Change the <PATH> to the value returned from the go env GOBIN command above.

```shell
provider_installation {
  dev_overrides {
      "registry.terraform.io/xenserver/xenserver" = "<PATH>"
  }
  direct {}
}
```

3. To compile the provider, run `go install .`. This will build the provider and put the provider binary in the <GOBIN> directory.

4. Local test with terraform command, you'll first need Terraform installed on your machine (see [Requirements](#requirements) above). Go to `examples/terraform-main/` folder, update the main.tf and run terraform commands:

```shell
terraform plan
terraform apply -auto-approve

// show state 
terraform state show xenserver_vm.vm1

// remove state
terraform state rm xenserver_vm.vm1

// import state with id
terraform import xenserver_vm.vm1 "9cf22238-ec83-1d5f-1632-db0c57be82dd"
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
