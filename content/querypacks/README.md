# cnspec Querypacks

Querypacks allow you to collect data from any type of system that you interrogate to create an inventory of systems, configuration, and their relationships.

## Getting Started

### Install cnspec

Before using these querypacks, install cnspec on your system:

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

## Available Querypacks

This folder contains a collection of querypacks for most common use-cases, including operating systems, services, and incident response (when you really want to collect critical data quickly).

## Run querypacks

```bash
cnspec scan {TARGET} -f {QUERYPACK}
```

Examples:

```bash
# Linux
cnspec scan local -f mondoo-linux-inventory.mql.yaml
```

After running a scan, cnspec displays the asset inventory.

To learn more about querypacks, check out our [docs](https://mondoo.com/docs/cnquery/cnquery-run-pack).

### Output Formats

Export results in different formats for integration with other tools:

```bash
# JSON output
cnspec scan local -o json > results.json

# Full detailed output
cnspec scan local -o full
```

## Join the community!

Join the [Mondoo Community GitHub Discussions](https://github.com/orgs/mondoohq/discussions) to collaborate on policy as code and security automation.

## Additional policies

Additional certified security and compliance policies can be found in the Policy Hub on Mondoo Platform. [Sign up for a free account](https://mondoo.com/pricing) to view the list of policies available.

## License

[Business Source License 1.1](../LICENSE)
