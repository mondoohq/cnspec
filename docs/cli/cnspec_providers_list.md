---
id: cnspec_providers_list
title: cnspec providers list
---


List all providers on the system

```bash
cnspec providers list [flags]
```

### Options

```
  -h, --help   help for list
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

