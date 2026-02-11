# cnquery Query Packs

Query packs are pre-built collections of queries written in [MQL (Mondoo Query Language)](https://mondoo.com/docs/mql/home/) that help you gather information from your infrastructure for asset inventory, incident response, and security assessments. These packs are designed to work with cnquery, the open-source, cloud-native tool that answers every question about your infrastructure.

## What are Query Packs?

Query packs contain structured queries that:

- **Inventory assets** across cloud providers, operating systems, and applications
- **Support incident response** by gathering critical system information
- **Assess security posture** through targeted data collection
- **Standardize data gathering** across different platforms and environments

Each query pack is tailored for specific platforms and use cases, making it easy to get started with infrastructure assessment without writing queries from scratch.

## Available Query Packs

### Asset Inventory Packs

- **AWS** (`mondoo-aws-inventory.mql.yaml`) - Comprehensive AWS account and resource inventory
- **Azure** (`mondoo-azure-inventory.mql.yaml`) - Azure subscription and service inventory
- **GCP** (`mondoo-gcp-inventory.mql.yaml`) - Google Cloud Platform resource discovery
- **Kubernetes** (`mondoo-kubernetes-inventory.mql.yaml`) - Container orchestration platform inventory
- **Linux** (`mondoo-linux-inventory.mql.yaml`) - Linux system and package inventory
- **macOS** (`mondoo-macos-inventory.mql.yaml`) - macOS system information gathering
- **Windows** (`mondoo-windows-inventory.mql.yaml`) - Windows system and application inventory
- **VMware** (`mondoo-vmware-inventory.mql.yaml`) - VMware infrastructure inventory
- **GitHub** (`mondoo-github-inventory.mql.yaml`) - GitHub organization and repository inventory
- **Shodan** (`mondoo-shodan-inventory.mql.yaml`) - Internet-facing asset discovery
- **DNS** (`mondoo-dns-inventory.mql.yaml`) - DNS configuration and record inventory
- **Email** (`mondoo-email-inventory.mql.yaml`) - Email system configuration
- **Slack** (`mondoo-slack-inventory.mql.yaml`) - Slack workspace inventory
- **Terraform** (`mondoo-terraform-inventory.mql.yaml`) - Infrastructure as code inventory

### Incident Response Packs

- **AWS** (`mondoo-aws-incident-response.mql.yaml`) - AWS security event investigation
- **Linux** (`mondoo-linux-incident-response.mql.yaml`) - Linux system forensics and analysis
- **macOS** (`mondoo-macos-incident-response.mql.yaml`) - macOS security incident analysis
- **Windows** (`mondoo-windows-incident-response.mql.yaml`) - Windows security investigation
- **Kubernetes** (`mondoo-kubernetes-incident-response.mql.yaml`) - Container security analysis
- **VMware** (`mondoo-vmware-incident-response.mql.yaml`) - VMware security assessment
- **GitHub** (`mondoo-github-incident-response.mql.yaml`) - GitHub security event analysis
- **Google Workspace** (`mondoo-googleworkplace-incident-response.mql.yaml`) - Workspace security investigation
- **Okta** (`mondoo-okta-incident-response.mql.yaml`) - Identity provider security analysis
- **OpenSSL** (`mondoo-openssl-incident-response.mql.yaml`) - SSL/TLS security assessment
- **SSL/TLS Certificates** (`mondoo-ssl-tls-certificate-incident-response.mql.yaml`) - Certificate security analysis

### Specialized Packs

- **Asset Count** (`mondoo-asset-count.mql.yaml`) - Simple asset counting across platforms
- **Windows Operational** (`mondoo-windows-operational-inventory.mql.yaml`) - Windows operational data

## Usage

### Basic Usage

Run a query pack against your infrastructure:

```bash
cnquery scan -f mondoo-aws-inventory.mql.yaml
```

### Target Specific Assets

Run against specific targets:

```bash
# Local system
cnquery scan local -f mondoo-linux-inventory.mql.yaml

# Remote SSH
cnquery scan ssh user@hostname -f mondoo-linux-incident-response.mql.yaml

# AWS account
cnquery scan aws -f mondoo-aws-inventory.mql.yaml

# Kubernetes cluster
cnquery scan k8s -f mondoo-kubernetes-inventory.mql.yaml
```

### Output Formats

Export results in different formats:

```bash
# JSON output
cnquery scan -f mondoo-aws-inventory.mql.yaml --output json

# YAML output
cnquery scan -f mondoo-aws-inventory.mql.yaml --output yaml

# Compact output
cnquery scan -f mondoo-aws-inventory.mql.yaml --output compact
```

## Query Pack Structure

Each query pack is a YAML file that contains:

- **Metadata**: Name, version, author, and licensing information
- **Platform filters**: Automatic targeting based on asset type
- **Queries**: MQL queries organized by purpose
- **Documentation**: Descriptions and usage guidance

Example structure:

```yaml
packs:
  - uid: example-pack
    name: Example Query Pack
    version: 1.0.0
    queries:
      - uid: example-query
        title: Example Query
        mql: asset.name
```

## Creating Custom Query Packs

1. **Start with an existing pack** as a template
2. **Define your queries** using MQL syntax
3. **Add appropriate filters** for target platforms
4. **Test thoroughly** across your target environments
5. **Document your pack** with clear descriptions

For MQL syntax and available resources, see the [MQL documentation](https://mondoo.com/docs/mql/home/).

## Contributing

We welcome contributions from the community! Query packs are maintained collaboratively with support from the Mondoo team.

### How to Contribute

1. **Fork the repository** and create a feature branch
2. **Add or modify query packs** following existing patterns
3. **Test your changes** against relevant target systems
4. **Submit a pull request** with a clear description of your changes

For detailed contribution guidelines, see our [Contributing Guide](https://github.com/mondoohq/.github/blob/master/CONTRIBUTING.md).

### Community Support

- **GitHub Discussions**: Join the [Mondoo Community](https://github.com/orgs/mondoohq/discussions) to collaborate on policy as code and security automation
- **Issues**: Report bugs or request features through GitHub Issues
- **Documentation**: Help improve documentation and examples

## License

Query packs are licensed under BUSL-1.1. See individual pack files for specific licensing information.

## Support

For questions about cnquery or query packs:

- 📚 [Documentation](https://mondoo.com/docs/)
- 💬 [Community Discussions](https://github.com/orgs/mondoohq/discussions)
- 🐛 [Report Issues](https://github.com/mondoohq/mql/issues)
