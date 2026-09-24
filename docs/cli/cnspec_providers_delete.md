---
id: cnspec_providers_delete
title: cnspec providers delete
---


Remove an installed provider from disk

### Synopsis

Remove an installed provider plugin from disk. The provider is
re-downloaded automatically the next time it's needed.

Use the special target "all" to remove every installed provider at once. Because
that wipes your whole provider footprint, it requires the --yes flag to confirm.

Examples:
  mql providers delete aws          # remove the aws provider
  mql providers delete all --yes    # remove every installed provider

```bash
cnspec providers delete <NAME> [flags]
```

### Options

```
  -h, --help   help for delete
      --yes    Confirm removal of all providers when using the 'all' target
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

* [cnspec providers](cnspec_providers.md)	 - Providers add connectivity to all assets

