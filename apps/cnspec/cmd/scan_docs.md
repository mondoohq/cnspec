Run a security scan on an asset based on one or more Mondoo policies.

To learn more, read [Get Started with cnspec](/cnspec/).

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
