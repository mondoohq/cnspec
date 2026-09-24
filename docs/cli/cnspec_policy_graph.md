---
id: cnspec_policy_graph
title: cnspec policy graph
---


Navigate policy bundle structure via graph commands

### Synopsis

Build and query a graph of policies, checks, frameworks, and controls from .mql.yaml bundle files.

### Options

```
  -h, --help   help for graph
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
* [cnspec policy graph callees](cnspec_policy_graph_callees.md)	 - Show what a node contains or references (outbound edges)
* [cnspec policy graph callers](cnspec_policy_graph_callers.md)	 - Show what references a node (inbound edges)
* [cnspec policy graph context](cnspec_policy_graph_context.md)	 - Show LLM-friendly context with YAML snippets
* [cnspec policy graph export](cnspec_policy_graph_export.md)	 - Export the full policy graph
* [cnspec policy graph paths](cnspec_policy_graph_paths.md)	 - Find paths between two nodes
* [cnspec policy graph reachable](cnspec_policy_graph_reachable.md)	 - Show all nodes reachable from a node
* [cnspec policy graph search](cnspec_policy_graph_search.md)	 - Search for nodes by name, title, or UID

