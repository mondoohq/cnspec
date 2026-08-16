# Mondoo GCP Security Terraform Pass Testdata

This directory contains terraform files that are used to test the Mondoo GCP Security policy. The terraform code provisions resources that are compliant with the security best practices defined in the Mondoo GCP Security policy pack.

## Setup

```shell
terraform init
```

## TF HCL Scanning

Run `cnspec` against the terraform files:

```shell
cnspec scan terraform hcl testdata/mondoo-gcp-security-tf-pass -f mondoo-gcp-security.mql.yaml
```

## TF Plan Scanning

```shell
terraform plan -var-file="terraform.tfvars" -out=tfplan.binary
terraform show -json tfplan.binary > tfplan.json
```

Then run `cnspec` against the generated plan:

```shell
cnspec scan terraform plan testdata/mondoo-gcp-security-tf-pass/tfplan.json -f mondoo-gcp-security.mql.yaml
```
