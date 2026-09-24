---
id: cnspec
title: cnspec
---


cnspec CLI

### Synopsis

cnspec is a cloud-native security testing tool for your entire infrastructure


### Options

```
      --api-proxy string        Set the proxy for communications with Mondoo Platform API
      --auto-update             Enable automatic provider installation and update (default true)
      --config string           Set config file path (default $HOME/.config/mondoo/mondoo.yml)
  -h, --help                    help for cnspec
      --log-level string        Set the log level: error, warn, info, debug, trace (default "info")
      --logging-config string   Path to a logging configuration file (YAML or JSON) that selects the log writer, level, and writer-specific options
      --strict                  Default MQL strict mode for policies that do not declare one: every link in an MQL chain must resolve
  -v, --verbose                 Enable verbose output
```

### SEE ALSO

* [cnspec aibom](cnspec_aibom.md)	 - Generate an AI bill of materials (AIBOM) for AI models across providers
* [cnspec discover](cnspec_discover.md)	 - Discover assets
* [cnspec framework](cnspec_framework.md)	 - Manage local and Mondoo Platform hosted compliance frameworks
* [cnspec login](cnspec_login.md)	 - Register with Mondoo Platform
* [cnspec logout](cnspec_logout.md)	 - Log out from Mondoo Platform
* [cnspec lsp](cnspec_lsp.md)	 - Launch the MQL Language Server
* [cnspec migrate](cnspec_migrate.md)	 - Migrate cnspec CLI configuration to the latest version
* [cnspec policy](cnspec_policy.md)	 - Manage local and upstream policies
* [cnspec providers](cnspec_providers.md)	 - Providers add connectivity to all assets
* [cnspec run](cnspec_run.md)	 - Run an MQL query
* [cnspec sbom](cnspec_sbom.md)	 - Experimental: Generate a software bill of materials (SBOM) for a given asset
* [cnspec scan](cnspec_scan.md)	 - Scan assets with one or more policies
* [cnspec serve](cnspec_serve.md)	 - Start cnspec in background mode
* [cnspec status](cnspec_status.md)	 - Verify access to Mondoo Platform
* [cnspec vault](cnspec_vault.md)	 - Manage vault environments
* [cnspec version](cnspec_version.md)	 - Display the cnspec version

