---
id: cnspec_providers_resources
title: cnspec providers resources
---


List resources or show resource details for a provider

### Synopsis

List all resources available in a provider, or show detailed field information
for a specific resource. The schema includes core and network resources.

Examples:
  cnspec providers resources aws              # list all resources
  cnspec providers resources aws --json       # list all resources as JSON
  cnspec providers resources aws aws.ec2.instance         # show resource details
  cnspec providers resources aws aws.ec2.instance --json  # show resource details as JSON

```bash
cnspec providers resources <provider> [<resource>] [flags]
```

### Options

```
  -h, --help   help for resources
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

* [cnspec providers](cnspec_providers.md)	 - Providers add connectivity to all assets

