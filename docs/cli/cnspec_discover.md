---
id: cnspec_discover
title: cnspec discover
---


Discover assets

### Synopsis

Discover assets defined by an inventory file's discovery targets/filters or via CLI parameters. Prints a per-platform asset count to stdout. Pass --output-full <path> to additionally write every discovered asset to a file; pick the file format with --output-format json|jsonl|yaml (default json). No queries are executed.

```bash
cnspec discover [flags]
```

### Options

```
  -h, --help                    help for discover
      --inventory-file string   Set the path to the inventory file
  -f, --output-format string    Format for --output-full: json (default), jsonl, or yaml. (default "json")
  -o, --output-full string      Write every discovered asset to this path. When empty, only the per-platform count summary is printed.
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

