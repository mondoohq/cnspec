---
id: cnspec_status
title: cnspec status
---

Verify access to Mondoo Platform

### Synopsis

Status sends a ping to Mondoo Platform to verify the credentials.

```bash
cnspec status [flags]
```

### Options

```
  -h, --help            help for status
  -o, --output string   Set output format. Accepts json or yaml.
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

- [cnspec](cnspec) - cnspec CLI
