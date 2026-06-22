WORKDIR ?= examples/terraform-main

SHELL := /bin/bash
default: testacc

# Run acceptance tests
.PHONY: testacc
testacc: provider ## make testacc
	source .env \
    && TF_ACC=1 go test -v $(TESTARGS) -timeout 60m ./xenserver/ \
    && TF_ACC=1 TEST_POOL=1 go test -v -run TestAccPoolResource -timeout 60m ./xenserver/

testpool: provider
	source .env \
    && TF_ACC=1 TEST_POOL=1 go test -v -run TestAccPoolResource -timeout 60m ./xenserver/

doc:  ## make doc for terraform provider documentation
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name xenserver

provider: go.mod  ## make provider
	if [ -z "$(GOBIN)" ]; then echo "GOBIN is not set" && exit 1; fi
	rm -f $(GOBIN)/terraform-provider-xenserver
	go mod tidy
	go install .
	md5sum $(GOBIN)/terraform-provider-xenserver

apply: .env provider  ## make apply
	cd $(WORKDIR) && \
    terraform plan && \
    terraform apply -auto-approve

apply_vm: .env provider  ## make apply_vm
	$(MAKE) WORKDIR=examples/vm-main apply

apply_pool: .env provider  ## make apply_pool
	$(MAKE) WORKDIR=examples/pool-main apply

show_state: .env  ## make show_state resource=xenserver_vm.vm
	@cd $(WORKDIR) && \
	if [ -z "$(resource)" ]; then echo "USAGE: make show_state resource=<>" && \
	echo "List available resources:" && echo "`terraform state list`" && exit 1; fi && \
	terraform state show $(resource)

import: .env  ## make import resource=xenserver_vm.vm id=vm-uuid
	@cd $(WORKDIR) && \
	if [ -z "$(resource)" ] || [ -z "$(id)" ]; then echo "USAGE: make import resource=<> id=<>"; exit 1; fi && \
	terraform import $(resource) $(id)

remove: .env  ## make remove resource=xenserver_vm.vm
	@cd $(WORKDIR) && \
	if [ -z "$(resource)" ]; then echo "USAGE: make remove resource=<>"; exit 1; fi && \
	terraform state rm $(resource)

destroy:
	cd $(WORKDIR) && \
    terraform destroy -auto-approve

destroy_vm:
	$(MAKE) WORKDIR=examples/vm-main destroy

destroy_pool:
	$(MAKE) WORKDIR=examples/pool-main destroy
