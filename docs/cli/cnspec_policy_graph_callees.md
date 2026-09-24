---
id: cnspec_policy_graph_callees
title: cnspec policy graph callees
---


Show what a node contains or references (outbound edges)

```bash
cnspec policy graph callees <uid> <path> [flags]
```

### Options

```
  -h, --help   help for callees
      --json   Output as JSON
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

* [cnspec policy graph](cnspec_policy_graph.md)	 - Navigate policy bundle structure via graph commands

