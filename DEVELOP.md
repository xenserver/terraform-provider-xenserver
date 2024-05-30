# Project Development Documentation

## Overview

This is the best practice document for XenServer terraform provider development, target to make contributors understand how to make changes, run local checks or tests, raise and merge PRs.

## Coding Standards

### Add new resource, data_source and function

1. When add a new component like [resource](https://developer.hashicorp.com/terraform/plugin/framework/resources), [data_source](https://developer.hashicorp.com/terraform/plugin/framework/data-sources) and [function](https://developer.hashicorp.com/terraform/plugin/framework/functions), create a file `<name>_<component_type>.go` under folder `xenserver/` and start coding. Note: For each new `resource`, requires a "id" field in the Schema.

2. For each new added component, add the according "NewXX" function into the return value of provider function `Resources`, `DataSources` or `Functions` under `xenserver/provider.go`.

3. For each new added component, create an acceptance test file `<name>_<component_type>_test.go` under folder `xenserver/`. The test configuration can be written together under `xenserver/test.config.go`.

4. For each new added component, requires to add an example for it under folder `examples/`.

- `provider`

    create a file `examples/provider/provider.tf` to show how to config and use this provider.

- `resource`
      
    create two files under folder `examples/resources/<resource_name>/`. `install.sh` to show how to import a existing resource and `resource.tf` to show how to configure with this resource.

- `data_source`

    create a file `data-source.tf` under folder `examples/data-sources/<data-source_name>/` to show how to configure with this data-source.

- `function`

    create a file `function.tf` under folder `examples/functions/<function_name>/` to show how to use this function. 

5. Generate new documents base on changes, run `go generate ./...`.

### Local Checking and Testing

Before push your commit, suggest to run below checks and tests to confirm the code quality first.

1. Format code with `gofmt -w -l xenserver/*.go` and run `golangci-lint` to make sure no Go lint error.

```shell
sudo docker run -it -v $(pwd):/app -w /app golangci/golangci-lint bash
golangci-lint run --config=/app/.golangci.yml
```

2. Run and pass the full suite of acceptance tests with `make testacc`.

3. Add new component configuration under `/examples/terraform-main/main.tf`, and run some manual tests(see [Prepare Terraform for local provider install](README.md)).

*Note:* Before running tests, the XenServer instance should be properly set up.

### Name rules

- component name, like resource, data-source, function, follow `xenserver_<name>`. eg.

```shell
xenserver_vm
```

- function name follow `Pascal`, eg.

```shell
func GetFirstTemplate(){}
```

- var name follow `Camel-Case`, eg.

```shell
var dataState VMResourceModel
```

## Development Process For Community Contributors

*Note:* Always Keep your local master branch up to date of master branch on [terraform-provider-xenserver](https://github.com/xenserver/terraform-provider-xenserver).

Preparation:

1. Clone the repository [terraform-provider-xenserver](https://github.com/xenserver/terraform-provider-xenserver) to local machine.
2. Fork the Github repository [terraform-provider-xenserver](https://github.com/xenserver/terraform-provider-xenserver). Add
it as another remote "github-fork" to local repository.
3. Prepare your local new commit. Make sure this commit already rebase latest master before next steps.

Steps:

1. Push commit to `github-fork` and raise PR from `github-fork` to [terraform-provider-xenserver](https://github.com/xenserver/terraform-provider-xenserver), the PR should contains the messages of acceptance tests result.
2. Contact the repository owner to help trigger the internal tests.
4. 2 approves needed before merger PR.

## Version Control

The repository owner will add a tag when decide to release a new version of terraform-provider-xenserver. Tag will trigger the Github actions to build a new package to Github release and terraform registry will pick up this new package.
