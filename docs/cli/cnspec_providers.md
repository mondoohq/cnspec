---
id: cnspec_providers
title: cnspec providers
---


Providers add connectivity to all assets

### Synopsis

Manage your providers. List and install new ones or update existing ones

```bash
cnspec providers [flags]
```

### Options

```
  -h, --help   help for providers
      --json   Output in JSON format
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
* [cnspec providers delete](cnspec_providers_delete.md)	 - Remove an installed provider from disk
* [cnspec providers info](cnspec_providers_info.md)	 - Show detailed information about one or more providers
* [cnspec providers install](cnspec_providers_install.md)	 - Install or update a provider
* [cnspec providers list](cnspec_providers_list.md)	 - List all providers on the system
* [cnspec providers resources](cnspec_providers_resources.md)	 - List resources or show resource details for a provider
* [cnspec providers update](cnspec_providers_update.md)	 - Update installed providers to their latest versions

