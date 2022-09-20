# cnspec

Welcome to cnspec, the cloud-native security assessment and testing tool for your entire fleet!

Here are a few examples of what it can do:

```
# scan the local system
cnspec scan local

# run a policy bundle file against a docker container
cnspec scan docker 14119a -f policy.mql.yaml

# open an interactive shell to an aws account
cnquery shell aws
> ec2.instances.all( detailedMonitoring == "enabled" )
```


## Quick Start

Please ensure you have the latest [Go 1.19.0+](https://golang.org/dl/) and latest [Protocol Buffers](https://github.com/protocolbuffers/protobuf/releases).  

Building:

```bash
# install all dependent tools
make prep 

# build and install cnquery
make build
make install
```

Some files in this repo are auto-generated. Whenever a proto or resource pack is changed, these will need to be rebuilt. Please re-run:

```bash
make cnspec/generate
```

## Development

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals 📄 docs
✨⭐🌟🎉 smaller or larger features 🐎 race condition
🌙 MQL 🌈 visual 🍏 fix tests 🎫 auth 🦅 falcon 🐳 container


## Legal

- **Copyright:** 2018-2022, Mondoo Inc, proprietary
- **Authors:** Christoph Hartmann, Dominik Richter

