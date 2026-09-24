---
id: cnspec_policy_info
title: cnspec policy info
---


Show more info about a policy from the connected space

```bash
cnspec policy info UID/MRN [flags]
```

### Options

```
  -f, --file string   Set the path to a local bundle file
  -h, --help          help for info
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

* [cnspec policy](cnspec_policy.md)	 - Manage local and upstream policies

