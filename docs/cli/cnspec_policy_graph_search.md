---
id: cnspec_policy_graph_search
title: cnspec policy graph search
---


Search for nodes by name, title, or UID

### Synopsis

Find policy graph nodes using multi-strategy search: exact name, prefix, or substring match across names, qualified names, and titles.

```bash
cnspec policy graph search <query> <path> [flags]
```

### Options

```
  -h, --help          help for search
      --impact int    Minimum impact score
      --json          Output as JSON
      --kind string   Filter by node kind (policy, check, group, query, framework, control)
      --limit int     Maximum results (default 50)
      --tag string    Filter by tag key
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

