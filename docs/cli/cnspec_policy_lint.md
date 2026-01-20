---
id: cnspec_policy_lint
title: cnspec policy lint
---

Lint a policy bundle

```bash
cnspec policy lint [path] [flags]
```

### Options

```
  -h, --help                 help for lint
  -o, --output string        Set output format: compact, sarif (default "cli")
      --output-file string   Set output file
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

- [cnspec policy](cnspec_policy) - Manage local and upstream policies
