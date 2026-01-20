---
id: cnspec_policy_upload
title: cnspec policy upload
---

Upload a policy to the connected space

```bash
cnspec policy upload my.mql.yaml [flags]
```

### Options

```
  -h, --help                    help for upload
      --no-lint                 Disable linting of the bundle before publishing.
      --policy-version string   Override the version of each policy in the bundle.
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
