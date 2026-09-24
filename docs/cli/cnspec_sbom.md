---
id: cnspec_sbom
title: cnspec sbom
---


Experimental: Generate a software bill of materials (SBOM) for a given asset

### Synopsis

Generate a software bill of materials (SBOM) for a given asset. The SBOM
is a representation of the asset's software components and their dependencies.

The following formats are supported:
- list (default)
- cnquery-json
- cyclonedx-json
- cyclonedx-xml
- spdx-json
- spdx-tag-value

Note this command is experimental and may change in the future.


```bash
cnspec sbom [flags]
```

### Options

```
      --annotation stringToString   Add an annotation to the asset in the form KEY=VALUE (default [])
      --asset-name string           User-override for the asset name
  -h, --help                        help for sbom
  -o, --output string               Set output format: json, cyclonedx-json, cyclonedx-xml, spdx-json, spdx-tag-value, table (default "list")
      --output-target string        Set output target to which the SBOM report will be written
      --with-cpes                   Generate CPEs for each component
      --with-evidence               Include evidence for each component
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

