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

For more installation options, see the [cnspec installation guide](https://mondoo.com/docs/cnspec/cnspec-adv-install/overview/).

### Verify installation

```bash
cnspec version
```

## Available Security Policies

Our comprehensive collection of security policies covers major platforms and services:

### Cloud Providers

- **AWS** - `mondoo-aws-security.mql.yaml` - Comprehensive AWS security baseline and best practices
- **Azure** - `mondoo-azure-security.mql.yaml` - Microsoft Azure security configuration and compliance checks
- **GCP** - `mondoo-gcp-security.mql.yaml` - Google Cloud Platform security assessment and hardening
- **OCI** - `mondoo-oci-security.mql.yaml` - Oracle Cloud Infrastructure security assessment

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
- **Proxmox VE Security** - `mondoo-proxmox-security.mql.yaml` - Proxmox Virtual Environment hypervisor, VM, and container security (CVE checks: `vulnerabilities/mondoo-proxmox-vulnerability.mql.yaml`)
- **Terraform Deprecations** - `terraform-deprecations.mql.yaml` - Detect deprecated Terraform constructs

### Network Devices

- **Arista EOS** - `mondoo-arista-eos-security.mql.yaml` - Arista EOS network device security hardening
- **Cisco IOS-XE** - `mondoo-cisco-iosxe-security.mql.yaml` - Cisco IOS-XE security configuration and hardening
- **Cisco IOS-XR** - `mondoo-cisco-iosxr-security.mql.yaml` - Cisco IOS-XR security configuration and hardening
- **Cisco NX-OS** - `mondoo-cisco-nxos-security.mql.yaml` - Cisco NX-OS security configuration and hardening
- **F5 BIG-IP** - `mondoo-bigip-security.mql.yaml` - F5 BIG-IP security configuration assessment
- **Juniper JunOS** - `mondoo-junos-security.mql.yaml` - Juniper JunOS network device security hardening
- **Palo Alto PAN-OS** - `mondoo-panos-security.mql.yaml` - Palo Alto Networks PAN-OS security assessment
- **Ubiquiti UniFi** - `mondoo-unifi-security.mql.yaml` - Ubiquiti UniFi network security assessment

### SaaS & Collaboration

- **GitHub Security** - `mondoo-github-security.mql.yaml` - GitHub repository and organization security
- **GitHub Best Practices** - `mondoo-github-best-practices.mql.yaml` - GitHub repository best practices
- **GitLab** - `mondoo-gitlab-security.mql.yaml` - GitLab security configuration assessment
- **Microsoft 365** - `mondoo-m365-security.mql.yaml` - Microsoft 365 security and compliance checks
- **Google Workspace** - `mondoo-google-workspace-security.mql.yaml` - Google Workspace security validation
- **Okta** - `mondoo-okta-security.mql.yaml` - Identity provider security assessment
- **Slack** - `mondoo-slack-security.mql.yaml` - Slack workspace security configuration
- **Snowflake** - `mondoo-snowflake-security.mql.yaml` - Snowflake data platform security assessment
- **Cloudflare** - `mondoo-cloudflare-security.mql.yaml` - Cloudflare security configuration assessment
- **Tailscale** - `mondoo-tailscale-security.mql.yaml` - Tailscale network security configuration

### Network & Infrastructure Services

- **DNS** - `mondoo-dns-security.mql.yaml` - DNS security and configuration checks
- **HTTP** - `mondoo-http-security.mql.yaml` - Web service security and header validation
- **TLS** - `mondoo-tls-security.mql.yaml` - SSL/TLS configuration and certificate validation
- **Email** - `mondoo-email-security.mql.yaml` - Email security configuration assessment

### Specialized Systems

- **Chef Infra Client** - `mondoo-chef-infra-client.mql.yaml` - Chef Infra Client security configuration
- **Chef Infra Server** - `mondoo-chef-infra-server.mql.yaml` - Chef Infra Server security configuration
- **MCP** - `mondoo-mcp-security.mql.yaml` - Model Context Protocol server security assessment
- **Phoenix PLCnext** - `mondoo-phoenix-plcnext-security.mql.yaml` - Industrial automation security
- **EDR** - `mondoo-edr-policy.mql.yaml` - Endpoint Detection and Response validation

> The latest version of the policies in this repository requires cnspec v8+

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
- **- Skip** - The check was skipped (not applicable to this system)

Each failed check includes:
- **Impact score** (0-100) indicating severity
- **Description** explaining why this check matters
- **Remediation steps** to fix the issue

### Output Formats

Export results in different formats for integration with other tools:

```bash
# JSON output
cnspec scan local -o json > results.json

# JUnit XML (for CI/CD integration)
cnspec scan local -o junit > results.xml

# Full detailed output
cnspec scan local -o full
```

## Policy Structure

Each policy file is a YAML document that contains security and operational checks written in MQL (Mondoo Query Language). The policies are structured as follows:

### Basic Structure

```yaml
policies:
  - uid: example-security-policy
    name: Example Security Policy
    version: 1.0.0
    authors:
      - name: Mondoo Security Team
        email: hello@mondoo.com
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

- **Metadata**: Each policy includes unique identifiers, versioning, and authorship information
- **Platform Filters**: Automatic targeting based on asset type (e.g., `asset.platform == "linux"`)
- **Security Checks**: MQL queries that validate security configurations and compliance requirements
- **Impact Scoring**: Risk assessment scoring from 0-100 to prioritize findings
- **Documentation**: Descriptions, remediation guidance, and references to security standards

### MQL Query Language

Policies use MQL to query system configurations, cloud resources, and application settings. MQL provides:

- **Resource Access**: Query files, processes, users, cloud resources, and more
- **Filtering**: Use `where()` to filter results based on specific criteria
- **Assertions**: Validate configurations meet security requirements
- **Cross-Platform**: Same query syntax works across different operating systems and cloud providers

For detailed MQL syntax and available resources, see the [MQL documentation](https://mondoo.com/docs/mql/).

### Example Policy Check

```yaml
checks:
  - uid: ssh-root-login-disabled
    title: Ensure SSH root login is disabled
    impact: 90
    mql: |
      sshd.config.params["PermitRootLogin"] == "no"
    docs:
      desc: |
        Direct root login via SSH should be disabled to prevent
        unauthorized access and encourage the use of sudo for
        administrative tasks.
      remediation: |
        Edit /etc/ssh/sshd_config and set:
        PermitRootLogin no

        Then restart the SSH service.
```

## Join the community!

Join the [Mondoo Community GitHub Discussions](https://github.com/orgs/mondoohq/discussions) to collaborate on policy as code and security automation.

## Additional policies

Additional certified security and compliance policies can be found on Mondoo Platform. [Sign up for a free account](https://mondoo.com/pricing) to view the list of policies available.

## License

[Business Source License 1.1](../LICENSE)
