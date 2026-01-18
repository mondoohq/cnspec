---
id: cnspec_vault_configure
title: cnspec vault configure
---

Configure a vault environment

### Synopsis

cnquery vault configure mondoo-client-vault --type linux-kernel-keyring

```
cnspec vault configure VAULTNAME [flags]
```

### Options

```
  -h, --help                    help for configure
      --inventory-file string   Set the path to the inventory file
      --option stringToString   addition vault connection options, multiple options via --option key=value (default [])
      --type string             possible values: aws-parameter-store | aws-secrets-manager | encrypted-file | gcp-berglas | gcp-secret-manager | hashicorp-vault | keyring | linux-kernel-keyring | memory | none
```

### Options inherited from parent commands

```
      --api-proxy string   Set proxy for communications with Mondoo API
      --auto-update        Enable automatic provider installation and update (default true)
      --config string      Set config file path (default $HOME/.config/mondoo/mondoo.yml)
      --log-level string   Set log level: error, warn, info, debug, trace (default "info")
  -v, --verbose            Enable verbose output
```

### SEE ALSO

- [cnspec vault](cnspec_vault) - Manage vault environments
