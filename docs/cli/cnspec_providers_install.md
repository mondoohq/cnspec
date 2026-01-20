---
id: cnspec_providers_install
title: cnspec providers install
---

Install or update a provider

```bash
cnspec providers install <NAME[@VERSION]> [flags]
```

### Options

```
  -f, --file string   install a provider via a file
  -h, --help          help for install
      --url string    install a provider via URL
```

### Options inherited from parent commands

```
      --api-proxy string   Set proxy for communications with Mondoo API
      --auto-update        Enable automatic provider installation and update (default true)
      --config string      Set config file path (default $HOME/.config/mondoo/mondoo.yml)
      --log-level string   Set log level: error, warn, info, debug, trace (default "info")
  -v, --verbose            Enable verbose output
```

### SEE ALSO

- [cnspec providers](cnspec_providers) - Providers add connectivity to all assets
