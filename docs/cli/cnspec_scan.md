---
id: cnspec_scan
title: cnspec scan
---

Run a security scan on an asset based on one or more Mondoo policies.

To learn more, read [Get Started with cnspec](/cnspec/).

### Synopsis

This command triggers a new policy-based scan on an asset. By default, cnspec scans the local system with the default [policies](/cnspec/cnspec-policies/) built specifically for the platform. If you [register cnspec with Mondoo](/cnspec/cnspec-adv-install/registration/), this command scans using the applicable [enabled policies](/security/posture/policies/).

```bash
cnspec scan local
```

You can also specify a local policy and run it without storing results in Mondoo Platform:

```bash
cnspec scan local --policy-bundle POLICYFILE.yaml --incognito
```

In addition, cnspec can scan assets remotely using SSH. By default, cnspec uses the operating system's SSH agent and SSH config to retrieve the credentials:

```bash
cnspec scan ssh ec2-user@52.51.185.215
```

```bash
cnspec scan ssh ec2-user@52.51.185.215:2222
```

### Examples: cloud

#### Scan AWS

```bash
cnspec scan aws --region us-east-1
```

To learn more, read [Assess AWS Security with cnspec](/cnspec/cloud/aws).

#### Scan Azure

```bash
cnspec scan azure --subscription SUBSCRIPTION_ID --group GROUP_NAME
```

To learn more, read [Assess Azure Security with cnspec](/cnspec/cloud/azure/).

#### Scan Google Cloud (GCP)

```bash
cnspec scan gcp project PROJECT_ID
```

To learn more, read [Assess Google Cloud Security with cnspec](/cnspec/cloud/gcp/).

#### Scan Kubernetes

```bash
cnspec scan k8s
```

```bash
cnspec scan k8s MANIFEST_FILE
```

To learn more, read [Assess Kubernetes Security with cnspec](/cnspec/cloud/k8s/).

#### Scan Oracle Cloud Infrastructure (OCI)

```bash
cnspec scan oci
```

To learn more, read [Assess Oracle Cloud Infrastructure (OCI) Security with cnspec](/cnspec/cloud/oci/).

### Examples: SaaS

#### Scan GitHub

```bash
export GITHUB_TOKEN=YOUR_PERSONAL_ACCESS_TOKEN
cnspec scan github repo ORG/REPO
```

To learn more, read [Assess GitHub Security with cnspec](/cnspec/saas/github/).

#### Scan GitLab

```bash
cnspec scan gitlab --group YOUR_GROUP_NAME --token YOUR_TOKEN
```

#### Scan Google Workspace

```bash
export GOOGLEWORKSPACE_CLOUD_KEYFILE_JSON=/home/user/my-project-6646123456789.json
cnspec scan google-workspace --customer-id 5amp13iD --impersonated-user-email admin@domain.com
```

To learn more, read [Assess Google Workspace Security with cnspec](/cnspec/saas/google_workspace/).

#### Scan Jira

```bash
cnspec scan atlassian jira --host HOST_URL --user USER@DOMAIN --user-token YOUR_TOKEN
```

#### Scan Microsoft 365 (M365)

```bash
cnspec scan ms365 --certificate-path certificate.combo.pem --tenant-id YOUR_TENANT_ID --client-id YOUR_CLIENT_ID
```

To learn more, read [Assess Microsoft 365 Security with cnspec](/cnspec/saas/m365/).

#### Scan Okta

```bash
cnspec scan okta --organization your_org.okta.com --token API_TOKEN
```

To learn more, read [Assess Okta Security with cnspec](/cnspec/saas/okta/).

#### Scan Slack

```bash
cnspec scan slack --token API_TOKEN
```

To learn more, read [Assess Slack Security with cnspec](/cnspec/saas/slack/).

### Examples: supply chain and containers

cnspec supports local containers and images as well as images in Docker registries.

#### Scan Docker

```bash
cnspec scan docker container b62b276baab6
```

```bash
cnspec scan docker image ubuntu:latest
```

#### Scan Harbor

```bash
cnspec scan container registry harbor.lunalectric.com
```

#### Scan ECR

```bash
cnspec scan container registry 123456789.dkr.ecr.us-east-1.amazonaws.com/repository
```

#### Scan GCR

```bash
cnspec scan gcp gcr PROJECT_ID
```

#### Scan Vagrant

```bash
cnspec scan vagrant HOST
```

#### Scan an inventory file

```bash
cnspec scan --inventory-file FILENAME
```

### Options

```
      --annotation stringToString     Add an annotation to the asset in the form KEY=VALUE (default [])
      --asset-name string             Override the asset name
      --collect-support-bundle        Collect a support bundle (debug logs, asset bundle, inventory, resolved policy, report, provider versions) for sharing with Mondoo support. By default writes to a timestamped directory in the current working dir; override with --support-bundle-dir.
      --detect-cicd                   Try to detect CI/CD environments. If detected, set the asset category to 'cicd' (default true)
  -h, --help                          help for scan
      --incognito                     Run in incognito mode. Do not report scan results to Mondoo Platform
      --inventory-file string         Set the path to the inventory file
      --inventory-format-ansible      Set the inventory format to Ansible
      --inventory-format-domainlist   Set the inventory format to domain list
  -j, --json                          Run the query and return the object in a JSON structure
  -o, --output string                 Set the output format: compact, csv, full, hdf, json, json-v1, json-v2, junit, ocsf-json, ocsf-parquet, report, sarif, summary, yaml, yaml-v1, yaml-v2 (default "compact")
      --output-target string          Set the output target for the asset report: an AWS SQS topic URL, a local file, or a local directory (with -o hdf, one OHDF file per asset; with the OCSF formats, one file per event class)
      --parallelism int               Set the number of assets to scan in parallel. Defaults to a per-provider value capped by the CPUs available on this machine. Use 1 for sequential
      --platform-id string            Select a specific target asset by providing its platform ID
      --policy strings                Specify policies to execute. This requires --policy-bundle. You can pass multiple policies using --policy POLICY
  -f, --policy-bundle strings         Set the path to a policy file. Supports local paths, s3:// URIs, and http(s):// URLs
      --props stringToString          Set custom values for properties (default [])
      --risk-threshold int            Set the risk threshold. Exit with status 1 if any risk meets or exceeds this value (default 101)
      --support-bundle-dir string     Directory to write the support bundle into. Only used when --collect-support-bundle is set. Defaults to ./cnspec-support-bundle-<timestamp>/.
      --trace-id string               Set a trace identifier
```

### Options inherited from parent commands

```
      --api-proxy string        Set the proxy for communications with Mondoo Platform API
      --auto-update             Enable automatic provider installation and update (default true)
      --config string           Set config file path (default $HOME/.config/mondoo/mondoo.yml)
      --log-level string        Set the log level: error, warn, info, debug, trace (default "info")
      --logging-config string   Path to a logging configuration file (YAML or JSON) that selects the log writer, level, and writer-specific options
      --strict                  Default MQL strict mode for policies that do not declare one: every link in an MQL chain must resolve
  -v, --verbose                 Enable verbose output
```

### SEE ALSO

* [cnspec](cnspec.md)	 - cnspec CLI

