# cnspec-policies

This project contains security and operational best-practice policies (as code) for use with [cnspec](https://github.com/mondoohq/cnspec). The policies are published at the [Open Security Registry](https://mondoo.com/registry).

We've organized them into these directories:

- [core](core) - Core policies contain baseline security and operational best-practice checks for various scan targets. Core policies are maintained by Mondoo and have strict quality requirements.
- [extra](extra) - Extra policies are a mix of community- and Mondoo-maintained policy bundles that are outside Mondoo's core support tier.
- [community](community) - Community policies are primarily maintained by the community with the support of the Mondoo team. Community policies may move to extra or core over time. 

> The latest version of the policies in this repository requires cnspec v8+

## Run policies

```bash
cnspec scan {TARGET} -f core/{POLICY_NAME}.mql.yaml
```

Examples:

```bash
# Linux
cnspec scan local -f core/mondoo-linux-security.mql.yaml

# macOS
cnspec scan local -f core/mondoo-macos-security.mql.yaml

# Windows
cnspec scan local -f core/mondoo-windows-security.mql.yaml
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

## Join the community!

Join the [Mondoo Community GitHub Discussions](https://github.com/orgs/mondoohq/discussions) to collaborate on policy as code and security automation.

## Additional policies

Additional certified security and compliance policies can be found in the Policy Hub on Mondoo Platform. [Sign up for a free account](https://mondoo.com/pricing) to view the list of policies available.

## License

[Business Source License 1.1](LICENSE)
