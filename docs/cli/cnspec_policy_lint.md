---
id: cnspec_policy_lint
title: cnspec policy lint
---


Lint one or more policy bundles

```bash
cnspec policy lint [path ...] [flags]
```

### Options

```
  -h, --help                                help for lint
  -o, --output string                       Set the output format: compact, sarif (default "cli")
      --output-file string                  Set the output file
      --require-strict-declaration strict   Warn about policies that don't declare strict. Off by default while content is migrated; becomes the default in v14.
      --strict-rule strings                 Treat warnings from these rule IDs as errors (repeatable). Use 'all' to promote every warning to an error.
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

