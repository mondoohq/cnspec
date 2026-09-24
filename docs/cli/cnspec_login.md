---
id: cnspec_login
title: cnspec login
---


Register with Mondoo Platform

### Synopsis


Log in to Mondoo Platform using a registration token. To pass in the token, use
the '--token' flag.

You can generate a new registration token on the Mondoo Dashboard. Go to
https://app.mondoo.com -> Space -> Settings -> Registration Token. Copy the token and pass it in
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
      --updates-url string          Set the updates URL for mql and provider updates
```

### Options inherited from parent commands

```
      --api-proxy string        Set the proxy for communications with Mondoo Platform API
      --auto-update             Enable automatic provider installation and update (default true)
      --config string           Set config file path (default $HOME/.config/mondoo/mondoo.yml)
      --log-level string        Set the log level: error, warn, info, debug, trace (default "info")
      --logging-config string   Path to a logging configuration file (YAML or JSON) that selects the log writer, level, and writer-specific options
      --strict                  Default MQL strict mode for policies that do not declare one: every link in an MQL chain must resolve
  -v, --verbose                 Enable verbose output
```

### SEE ALSO

* [cnspec](cnspec.md)	 - cnspec CLI

