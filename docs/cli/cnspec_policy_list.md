---
id: cnspec_policy_list
title: cnspec policy list
---

List enabled policies in the connected space

```bash
cnspec policy list [-f bundle] [flags]
```

### Options

```
  -a, --all           list all policies, not only the enabled ones (applicable only for upstream)
  -f, --file string   a local bundle file
  -h, --help          help for list
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
