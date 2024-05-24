SHELL := /bin/bash
default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	source .env && TF_ACC=1 go test ./xenserver/ -v  $(TESTARGS) -timeout 120m
