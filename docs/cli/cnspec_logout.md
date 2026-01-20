---
id: cnspec_logout
title: cnspec logout
---

Log out from Mondoo Platform

### Synopsis

This process also revokes Mondoo Platform service account to
ensure the credentials cannot be used in the future.

```bash
cnspec logout [flags]
```

### Options

```
      --force   Force re-authentication
  -h, --help    help for logout
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
