---
id: cnspec_serve
title: cnspec serve
---

Start cnspec in background mode

```bash
cnspec serve [flags]
```

### Options

```
  -h, --help                    help for serve
      --inventory-file string   Set the path to the inventory file
      --splay int               randomize the timer by up to this many minutes (default 60)
      --timer int               scan interval in minutes (default 60)
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
