---
id: cnspec_login
title: cnspec login
---

Register with Mondoo Platform

### Synopsis

Log in to Mondoo Platform using a registration token. To pass in the token, use
the '--token' flag.

You can generate a new registration token on the Mondoo Dashboard. Go to
https://console.mondoo.com -> Space -> Settings -> Registration Token. Copy the token and pass it in
using the '--token' argument.

You remain logged in until you explicitly log out using the 'logout' subcommand.

```bash
cnspec login [flags]
```

### Options

```
      --annotation stringToString   Set the client annotations (default [])
      --api-endpoint string         Set the Mondoo API endpoint
  -h, --help                        help for login
      --name string                 Set asset name
      --splay int                   Randomize the timer by up to this many minutes
      --timer int                   Set the scan interval in minutes
  -t, --token string                Set a client registration token
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
