# cnspec Security Policies

Security and operational best-practice policies (as code) for use with [cnspec](https://github.com/mondoohq/cnspec), the open-source security scanner that assesses your entire infrastructure using policy as code.

## Getting Started

### Install cnspec

Before using these policies, install cnspec on your system:

```bash
# macOS / Linux (using Homebrew)
brew install mondoohq/mondoo/cnspec

# Windows (using Chocolatey)
choco install cnspec

# Or download directly from GitHub releases
# https://github.com/mondoohq/cnspec/releases
```

For more installation options, see the [cnspec installation guide](https://mondoo.com/docs/cnspec/install/overview).

### Verify installation

```bash
cnspec version
```

## Available Security Policies

Our comprehensive collection of security policies covers major platforms and services:

### Cloud Providers

- **Alibaba Cloud** - `mondoo-alibaba-security.mql.yaml` - Secure Alibaba Cloud identity, storage, compute, database, registry, serverless, key management, and network resources
- **AWS** - `mondoo-aws-security.mql.yaml` - Comprehensive AWS security baseline and best practices
- **Azure** - `mondoo-azure-security.mql.yaml` - Microsoft Azure security configuration and compliance checks
- **DigitalOcean** - `mondoo-digitalocean-security.mql.yaml` - DigitalOcean Droplets, Databases, Load Balancers, DOKS, Spaces, and App Platform security
- **GCP** - `mondoo-gcp-security.mql.yaml` - Google Cloud Platform security assessment and hardening
- **Hetzner Cloud** - `mondoo-hetzner-security.mql.yaml` - Hetzner Cloud servers, firewalls, load balancers, certificates, and IP address security
- **OCI** - `mondoo-oci-security.mql.yaml` - Oracle Cloud Infrastructure security assessment
- **OpenStack** - `mondoo-openstack-security.mql.yaml` - OpenStack security across Keystone, Nova, Neutron, Cinder, Glance, Swift, and Octavia
- **STACKIT** - `mondoo-stackit-security.mql.yaml` - Secure STACKIT compute, storage, databases, networking, and IAM

### Operating Systems

- **Linux** - `mondoo-linux-security.mql.yaml` - Linux system hardening and security configuration
- **Linux Workstation** - `mondoo-linux-workstation-security.mql.yaml` - Security baseline for Linux desktop and workstation environments
- **Linux Operational** - `mondoo-linux-operational-policy.mql.yaml` - Linux operational best practices
- **Linux SNMP** - `mondoo-linux-snmp-policy.mql.yaml` - SNMP security configuration for Linux systems
- **FreeBSD** - `mondoo-freebsd-security.mql.yaml` - FreeBSD security hardening and configuration
- **macOS** - `mondoo-macos-security.mql.yaml` - macOS security baseline and configuration management
- **Windows Server** - `mondoo-windows-security.mql.yaml` - Windows security hardening and compliance validation
- **Windows Workstation** - `mondoo-windows-workstation-security.mql.yaml` - Security baseline for Windows desktop and workstation environments
- **Windows 11 Compatibility** - `mondoo-windows-11-compatibility.mql.yaml` - Windows 11 upgrade readiness checks

### Container & Infrastructure

- **Dockerfile Security** - `mondoo-dockerfile-security.mql.yaml` - Container security and image vulnerability assessment
- **Dockerfile Best Practices** - `mondoo-dockerfile-best-practices.mql.yaml` - Dockerfile authoring best practices
- **Kubernetes Security** - `mondoo-kubernetes-security.mql.yaml` - Container orchestration security and RBAC validation
- **Kubernetes Best Practices** - `mondoo-kubernetes-best-practices.mql.yaml` - Kubernetes operational best practices
- **Nutanix** - `mondoo-nutanix-security.mql.yaml` - Secure Nutanix clusters, hosts, virtual machines, identity, and storage managed through Prism Central
- **Portainer** - `mondoo-portainer-security.mql.yaml` - Secure Portainer authentication, RBAC, user privileges, Edge trust, and connection settings
- **Kubernetes Kyverno Integration** - `mondoo-kubernetes-kyverno.mql.yaml` - Kyverno policy, PolicyReport, OpenReports, PolicyException, and Mondoo mapping visibility
- **Proxmox VE Security** - `mondoo-proxmox-security.mql.yaml` - Proxmox Virtual Environment hypervisor, VM, and container security
- **VMware vSphere** - `mondoo-vmware-vsphere.mql.yaml` - Security baseline for virtual machines running on VMware vSphere
- **VMware ESXi** - `mondoo-vmware-vsphere-esxi.mql.yaml` - VMware ESXi host hardening and configuration
- **Terraform Deprecations** - `terraform-deprecations.mql.yaml` - Detect deprecated Terraform constructs

### Network Devices

- **Arista EOS** - `mondoo-arista-eos-security.mql.yaml` - Arista EOS network device security hardening
- **Cisco IOS-XE** - `mondoo-cisco-iosxe-security.mql.yaml` - Cisco IOS-XE security configuration and hardening
- **Cisco IOS-XR** - `mondoo-cisco-iosxr-security.mql.yaml` - Cisco IOS-XR security configuration and hardening
- **Cisco NX-OS** - `mondoo-cisco-nxos-security.mql.yaml` - Cisco NX-OS security configuration and hardening
- **F5 BIG-IP** - `mondoo-bigip-security.mql.yaml` - F5 BIG-IP security configuration assessment
- **Fortinet FortiOS** - `mondoo-fortios-security.mql.yaml` - Fortinet FortiOS firewall and FortiGate appliance security
- **Juniper JunOS** - `mondoo-junos-security.mql.yaml` - Juniper JunOS network device security hardening
- **MikroTik RouterOS** - `mondoo-mikrotik-security.mql.yaml` - Secure MikroTik RouterOS routers, switches, and access points
- **Palo Alto PAN-OS** - `mondoo-panos-security.mql.yaml` - Palo Alto Networks PAN-OS security assessment
- **Ubiquiti UniFi** - `mondoo-unifi-security.mql.yaml` - Ubiquiti UniFi network security assessment

### SaaS & Collaboration

- **Atlassian** - `mondoo-atlassian-security.mql.yaml` - Detect high and critical security issues in Atlassian Cloud organizations, Jira projects, and Confluence spaces
- **Databricks** - `mondoo-databricks-security.mql.yaml` - Secure Databricks accounts, workspaces, clusters, and access controls
- **GitHub Security** - `mondoo-github-security.mql.yaml` - GitHub repository and organization security
- **GitHub Best Practices** - `mondoo-github-best-practices.mql.yaml` - GitHub repository best practices
- **GitLab** - `mondoo-gitlab-security.mql.yaml` - GitLab security configuration assessment
- **Microsoft 365** - `mondoo-m365-security.mql.yaml` - Microsoft 365 security and compliance checks
- **Google Workspace** - `mondoo-google-workspace-security.mql.yaml` - Google Workspace security validation
- **Grafana** - `mondoo-grafana-security.mql.yaml` - Grafana observability platform security configuration
- **Okta** - `mondoo-okta-security.mql.yaml` - Identity provider security assessment
- **Slack** - `mondoo-slack-security.mql.yaml` - Slack workspace security configuration
- **Snowflake** - `mondoo-snowflake-security.mql.yaml` - Snowflake data platform security assessment
- **Cloudflare** - `mondoo-cloudflare-security.mql.yaml` - Cloudflare security configuration assessment
- **Tailscale** - `mondoo-tailscale-security.mql.yaml` - Tailscale network security configuration
- **Vercel** - `mondoo-vercel-security.mql.yaml` - Secure Vercel projects and teams by enforcing deployment protection, firewall, secret hygiene, and credential controls

### Databases

- **MariaDB** - `mondoo-mariadb-security.mql.yaml` - Harden MariaDB server networking, authentication, encryption, and audit logging
- **MongoDB Atlas** - `mondoo-mongodbatlas-security.mql.yaml` - Secure MongoDB Atlas access control, network exposure, encryption, backup, and audit logging across organizations and projects
- **MySQL** - `mondoo-mysql-security.mql.yaml` - Harden MySQL server networking, authentication, privileges, and audit logging
- **PostgreSQL** - `mondoo-postgresql-security.mql.yaml` - Harden PostgreSQL server transport, host-based authentication, and audit logging

### Network & Infrastructure Services

- **DNS** - `mondoo-dns-security.mql.yaml` - DNS security and configuration checks
- **HTTP** - `mondoo-http-security.mql.yaml` - Web service security and header validation
- **TLS** - `mondoo-tls-security.mql.yaml` - SSL/TLS configuration and certificate validation
- **Email** - `mondoo-email-security.mql.yaml` - Email security configuration assessment
- **NextDNS** - `mondoo-nextdns-security.mql.yaml` - Secure NextDNS profiles by enforcing threat protection, filtering integrity, and query logging

### Specialized Systems

- **AI Security** - `mondoo-ai-security.mql.yaml` - Enforce approved AI coding agents, IDE extensions, MCP servers, and local LLM runtimes
- **Chef Infra Client** - `mondoo-chef-infra-client.mql.yaml` - Chef Infra Client security configuration
- **Chef Infra Server** - `mondoo-chef-infra-server.mql.yaml` - Chef Infra Server security configuration
- **MCP** - `mondoo-mcp-security.mql.yaml` - Model Context Protocol server security assessment
- **vLLM** - `mondoo-vllm-security.mql.yaml` - vLLM inference server HTTP exposure and endpoint hardening
- **Phoenix PLCnext** - `mondoo-phoenix-plcnext-security.mql.yaml` - Industrial automation security
- **EDR** - `mondoo-edr-policy.mql.yaml` - Endpoint Detection and Response validation
- **Shodan** - `mondoo-shodan.mql.yaml` - Shodan exposure assessment for hosts, networks, and IP addresses

> These policies track the cnspec release in this repository. Run a current cnspec (`cnspec version`); checks are written against the provider resources shipped with it, so an older binary may not be able to compile every check.

## Query packs

Alongside the policies, [`querypacks/`](querypacks) holds query packs: bundles that **collect data without scoring it**. Where a policy answers "is this configured safely", a query pack answers "what is out there" — asset inventory and incident-response data collection, per platform.

```bash
cnspec scan local -f querypacks/mondoo-linux-inventory.mql.yaml
```

## Infrastructure as Code

Most cloud checks carry `variants:`, so the same control is enforced against a live cloud account **and** against the code that provisions it. A check with variants runs whichever one matches the asset being scanned: the runtime API, Terraform HCL source, a `terraform plan` JSON file, a `terraform.tfstate`, a CloudFormation template, or a Bicep file.

```bash
cnspec scan terraform ./infrastructure -f mondoo-aws-security.mql.yaml
cnspec scan terraform plan  tfplan.json      -f mondoo-aws-security.mql.yaml
cnspec scan terraform state terraform.tfstate -f mondoo-aws-security.mql.yaml
```

This means a misconfiguration can be caught in a pull request rather than after it reaches production, with no separate policy to maintain.

## Run policies

```bash
cnspec scan {TARGET} -f {POLICY_NAME}.mql.yaml
```

Examples:

```bash
# Linux
cnspec scan local -f mondoo-linux-security.mql.yaml

# macOS
cnspec scan local -f mondoo-macos-security.mql.yaml

# Windows
cnspec scan local -f mondoo-windows-security.mql.yaml
```

With the Open Security Registry

```bash
cnspec scan {TARGET} --policy mondoohq/{POLICY_UID}
```

Examples:

```bash
# Linux
cnspec scan local --policy mondoohq/mondoo-linux-security

# macOS
cnspec scan local --policy mondoohq/mondoo-macos-security

# Windows
cnspec scan local --policy mondoohq/mondoo-windows-security
```

## Understanding Scan Results

After running a scan, cnspec displays results showing which checks passed or failed:

- **✓ Pass** - The check passed; the system meets the security requirement
- **✕ Fail** - The check failed; action is needed to remediate the issue
- **! Error** - The check encountered an error during execution
- **. Skipped** - The check was skipped because its `filters:` did not match this asset

Each failed check includes:
- **Impact score** (0-100) indicating severity
- **Description** explaining why this check matters
- **Remediation steps** to fix the issue

### Output Formats

Export results in different formats for integration with other tools:

```bash
# JSON output
cnspec scan local -o json > results.json

# SARIF (GitHub code scanning and other security dashboards)
cnspec scan local -o sarif > results.sarif

# JUnit XML (for CI/CD integration)
cnspec scan local -o junit > results.xml

# Full detailed output
cnspec scan local -o full
```

The full set is `compact` (the default), `csv`, `full`, `json`, `json-v1`, `json-v2`, `junit`, `report`, `sarif`, `summary`, `yaml`, `yaml-v1`, and `yaml-v2`. Run `cnspec scan --help` for the current list.

## Policy Structure

Each policy file is a YAML document that contains security and operational checks written in MQL (Mondoo Query Language). The policies are structured as follows:

### Basic Structure

```yaml
policies:
  - uid: example-security-policy
    name: Example Security Policy
    version: 1.0.0
    license: BUSL-1.1
    tags:
      mondoo.com/category: security
      mondoo.com/platform: linux
    require:
      - provider: os
    authors:
      - name: Mondoo, Inc.
        email: hello@mondoo.com
    summary: Secure example Linux accounts and shell configuration
    groups:
      - title: Security Configuration
        filters: asset.platform == "linux"
        checks:
          - uid: example-check
            title: Example Security Check
            impact: 80
            mql: |
              users.where(name == "root").list {
                shell != "/bin/bash"
              }
```

### Key Components

- **Metadata**: Unique identifier, version, license, and authorship
- **`summary`**: A one-line description, 130 characters or fewer, shown in policy listings
- **`tags`**: `mondoo.com/category` and `mondoo.com/platform` are required; `cnspec policy lint` warns without them
- **`require`**: The providers the policy needs, so cnspec can install them on demand
- **Platform Filters**: Which assets a check applies to (`asset.platform == "linux"`). This is asset selection, not check logic
- **Security Checks**: MQL queries that validate security configurations and compliance requirements
- **Impact Scoring**: Risk assessment scoring from 0-100 to prioritize findings
- **Documentation**: Descriptions, audit steps, remediation guidance, and compliance-framework tags

### MQL Query Language

Policies use MQL to query system configurations, cloud resources, and application settings. MQL provides:

- **Resource Access**: Query files, processes, users, cloud resources, and more
- **Filtering**: Use `where()` to filter results based on specific criteria
- **Assertions**: Validate configurations meet security requirements
- **Cross-Platform**: Same query syntax works across different operating systems and cloud providers

For detailed MQL syntax and available resources, see the [MQL documentation](https://mondoo.com/docs/mql).

### Example Policy Check

Every check documents all three of `desc`, `audit`, and `remediation`. `desc` explains what and why, `audit` gives an operator a way to verify the finding by hand, and `remediation` lists a fix per management surface:

````yaml
checks:
  - uid: ssh-root-login-disabled
    title: Ensure SSH root login is disabled
    impact: 90
    mql: |
      sshd.config.params["PermitRootLogin"] == "no"
    docs:
      desc: |
        Direct root login over SSH should be disabled to prevent unauthorized
        access and to require administrators to authenticate as themselves
        before escalating with sudo, which keeps actions attributable.
      audit: |
        Run the following and confirm it returns `no`:

        ```bash
        sshd -T | grep -i permitrootlogin
        ```
      remediation:
        - id: cli
          desc: |
            Set the option and reload the service:

            ```bash
            sudo sed -i 's/^#*PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
            sudo systemctl reload sshd
            ```
        - id: ansible
          desc: |
            ```yaml
            - name: Disable SSH root login
              ansible.builtin.lineinfile:
                path: /etc/ssh/sshd_config
                regexp: '^#*PermitRootLogin'
                line: 'PermitRootLogin no'
              notify: reload sshd
            ```
````

## Contributing a check

Policy changes are validated automatically, so the fastest path is to run the same checks CI does before opening a pull request:

```bash
cnspec policy lint mondoo-linux-security.mql.yaml   # must pass; run it first
make test/content                                   # lint + bundle scans + compliance mappings
```

Two documents cover the rest:

- **[`CLAUDE.md`](CLAUDE.md)** — the authoring rules: bundle structure, impact bands, UID conventions, the shape of `desc`/`audit`/`remediation`, compliance tagging, IaC variants, and the MQL behaviors that return a wrong verdict without erroring.
- **[`validation/README.md`](validation/README.md)** — every check that runs against this directory, what each one proves, when CI runs it, and how to run it yourself.

## Join the community!

Join the [Mondoo Community GitHub Discussions](https://github.com/orgs/mondoohq/discussions) to collaborate on policy as code and security automation.

## Additional policies

Additional certified security and compliance policies can be found on Mondoo Platform. [Sign up for a free account](https://mondoo.com/pricing) to view the list of policies available.

## License

[Business Source License 1.1](../LICENSE)
