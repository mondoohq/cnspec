---
id: cnspec_run
title: cnspec run
---


Run an MQL query

### Synopsis

Run an MQL query on the CLI and display its results.

```bash
cnspec run [flags]
```

### Options

```
      --ast                     Parse the query and return the abstract syntax tree (AST)
  -c, --command string          MQL query to execute
      --exit-1-on-failure       Exit with error code 1 if one or more query results fail
  -h, --help                    help for run
      --info                    Parse the query and provide information about it
      --inventory-file string   Set the path to the inventory file
  -j, --json                    Run the query and return the object in a JSON structure
      --parse                   Parse the query and return the logical structure
      --platform-id string      Select a specific target asset by providing its platform ID
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

