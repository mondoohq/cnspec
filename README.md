# cnspec

Cloud Native Security and Compliance Auditing Framework

cnspec is a cloud-native solution to test the security of your entire fleet. It finds vulnerabilities and misconfigurations on all systems in your infrastructure: cloud accounts, Kubernetes, containers, services, VMs, APIs, and more.

cnspec is a powerful policy engine built on [cnquery](https://github.com/mondoohq/cnquery), and comes with lots of policies out of the box. It is very simple to use:

```bash
cnspec scan local
``` 


## Installation

Please ensure you have the latest [Go 1.19.0+](https://golang.org/dl/) and latest [Protocol Buffers](https://github.com/protocolbuffers/protobuf/releases).  

Building:

```bash
# install all dependent tools
make prep 

# go install cnspec
export GOPRIVATE="github.com/mondoohq,go.mondoo.com"
make cnspec/install
```

Some files in this repo are auto-generated. Whenever a proto or resource pack is changed, these will need to be rebuilt. Please re-run:

```bash
make cnspec/generate
```

## Scan a system

Use `scan` to check your system for misconfigurations and vulnerabilities. 

This command evaluates the security of your local machine and tells you how to improve it:

```bash
cnspec scan local
```

You can also specify other targets to scan. These are examples:

```bash
# to scan a docker image:
cnspec scan docker image ubuntu:22.04

# to scan an aws account using the local AWS config
cnspec scan aws

# to scan a kubernetes cluster via your local kubectl config
cnspec scan k8s
```

##  Policies

cnspec comes with policies for most systems out of the box. For each target, it chooses any available default policy and runs it locally.

To explore more policies, visit our [cnspec-policies](https://github.com/mondoohq/cnspec-policies) GitHub repo.


###  Custom policies

You can write custom policies. A policy is simply a YAML file that lets you express any security rule or best practice for your fleet. 

The best place to start is to look at one of our example policies. You can find them in this repository. For example: `policy/examples/example.mql.yaml`. To run a local policy:

```bash
cnspec scan local --policy-bundle policy/examples/example.mql.yaml
```


## Interactive shell

The easiest way to explore custom assertions in cnspec is to use our interactive shell. This will help you understand the queries that are used in policies and write custom queries as well. It’s also a great way to interact with a system on the fly.

```bash
cnspec shell local
```

The shell uses auto-complete which makes it very easy to explore. Once inside the shell, you can enter MQL assertions like this:

```coffeescript
> ports.listening.none( port == 23 )
```

To find out more use the `help` command. To exit either hit CTRL+D or type `exit`.




## Distributing cnspec across your fleet

The easiest way to ensure your entire fleet is secure is to use share policies across your fleet. This can be done via the Query Hub.

This creates a secure private environment that keeps data about your assets in your own account. It makes it very easy for all nodes to report on policies and define custom exceptions for your fleet.

To use the Query Hub, run:

```bash
cnspec auth login
```

Once set up, you can scan the asset via:

```bash
cnspec scan local
```

To add custom policies, you can now upload policies via:

```bash
cnspec policy upload mypolicy.mql.yaml
```


## Where to go from here

There are so many things cnspec can do! From testing your entire fleet for vulnerabilities to gathering information about it and creating reports for auditors. With its custom policies cnspec can scan any component you care about!

Explore our:
- Policy Marketplace
- [Policy as Code](https://mondoo.com/docs/tutorials/mondoo/policy-as-code/)
- [MQL introduction](https://mondoohq.github.io/mql-intro/index.html)
- [MQL resource packs](https://mondoo.com/docs/references/mql/)
- [cnquery](https://github.com/mondoohq/cnquery), our open source, cloud-native asset inventory
- Using cnspec with Mondoo

Our goal is to secure all layers of your infrastructure. Join our [community](https://github.com/orgs/mondoohq/discussions) today and let’s grow it together!


## Development

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals 📄 docs  
✨⭐🌟🎉 smaller or larger features 🐎 race condition  
🌙 MQL 🌈 visual 🍏 fix tests 🎫 auth 🦅 falcon 🐳 container  


## Legal

- **Copyright:** 2018-2022, Mondoo Inc, proprietary
- **Authors:** Christoph Hartmann, Dominik Richter

