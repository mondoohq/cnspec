---
id: cnspec_policy_init
title: cnspec policy init
---

Create an example policy bundle

### Synopsis

Create an example policy bundle that you can use as a starting point. If you don't provide a filename, cnspec uses `example-policy.mql.yml`.

```bash
cnspec policy init [path] [flags]
```

### Options

```
  -h, --help   help for init
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
