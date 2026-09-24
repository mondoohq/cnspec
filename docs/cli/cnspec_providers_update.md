---
id: cnspec_providers_update
title: cnspec providers update
---


Update installed providers to their latest versions

### Synopsis

Update installed providers to their latest versions.

With no arguments, every installed provider is updated. Pass one or more
provider names to update just those. Providers already on the latest version
are skipped, and naming a provider that isn't installed is reported and skipped
rather than treated as an error.

--channel resolves this one command from a different release channel, without
changing any configuration. On a stable install that is how you try a
pre-release provider: the override lasts for this command only, so the next
update goes back to whatever is configured.

Examples:
  mql providers update              # update every installed provider
  mql providers update aws          # update just the aws provider
  mql providers update aws gcp      # update the aws and gcp providers
  mql providers update aws --channel preview   # one-off, from the pre-release track

```bash
cnspec providers update [<NAME>...] [flags]
```

### Options

```
      --channel string   Release channel to resolve from: stable or preview (default: the configured update_channel, or the channel this build belongs to)
  -h, --help             help for update
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

* [cnspec providers](cnspec_providers.md)	 - Providers add connectivity to all assets

