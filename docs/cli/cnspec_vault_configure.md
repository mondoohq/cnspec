---
id: cnspec_vault_configure
title: cnspec vault configure
---


Configure a vault environment

### Synopsis



mql vault configure mondoo-client-vault --type linux-kernel-keyring



```bash
cnspec vault configure VAULTNAME [flags]
```

### Options

```
  -h, --help                    help for configure
      --inventory-file string   Set the path to the inventory file
      --option stringToString   Set additional vault connection options (use --option key=value for multiple) (default [])
      --type string             Set the vault type. Possible values: aws-parameter-store | aws-secrets-manager | encrypted-file | gcp-berglas | gcp-secret-manager | hashicorp-vault | keyring | linux-kernel-keyring | memory | none
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

* [cnspec vault](cnspec_vault.md)	 - Manage vault environments

