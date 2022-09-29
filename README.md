# cnspec

**Cloud Native Security and Compliance Auditing Framework**

`cnspec` is a cloud-native solution to assess the security and compliance of your business-critical infrastructure. Using [policy as code](https://mondoo.com/policy-as-code/), `cnspec` finds vulnerabilities and misconfigurations on all systems in your infrastructure including: public and private cloud environments, Kubernetes clusters, containers, container registries, servers and endpoints, SaaS products, infrastructure as code, APIs, and more.

`cnspec` is a powerful policy engine built on [`cnquery`](https://github.com/mondoohq/cnquery), and comes configured with default security policies that run right out of the box. It is both fast and simple to use!

```bash
cnspec scan local
```

**Example output**
```bash
...

Controls:
✓ Pass:  Disable Media Sharing
✓ Pass:  Do not enable the "root" account
✓ Pass:  Disable Bluetooth Sharing
✕ Fail:  Enable security auditing
✓ Pass:  Enable Firewall
...
✕ Fail:  Ensure Firewall is configured to log
✓ Pass:  Ensure nfs server is not running.
✓ Pass:  Disable Content Caching
✕ Fail:  Ensure AirDrop Is Disabled
✓ Pass:  Control access to audit records


Summary
========================

Target:     user-macbook-pro
Score:      A    80/100     (100% completed)
✓ Passed:   ███████████ 70% (21)
✕ Failed:   ███ 17% (5)
! Errors:   ██ 13% (4)
» Skipped:  0% (0)

Policies:
A  80  macOS Security Baseline by Mondoo
```

## Installation

### Dependencies

Before building from source, make sure you have the following dependencies installed:

- [Go 1.19.0+](https://golang.org/dl/)
- [Protocol Buffers v21+](https://github.com/protocolbuffers/protobuf/releases)

On macOS systems with homebrew run: `brew install go@1.19 protobuf`

### Build & Install

To build and install cnspec via Go, run:

```bash
export GOPRIVATE="github.com/mondoohq,go.mondoo.com"
make cnspec/install
```

### Development

Whenever you change protos or other auto-generated files, you will need to regenerate files for the compiler. To do this, make sure you have the necessary tools installed (e.g. protobuf):

```bash
make prep
```

Then, whenever you make changes, just run:

```bash
make cnspec/generate
```

This will generate and update all required files for the build. At this point you can `make cnspec/install` again as outlined above.

## Run a scan

Use the `cnspec scan` subcommand to check local and remote targets for misconfigurations and vulnerabilities.

### Local scan

This command evaluates the security of your local machine:

```bash
cnspec scan local
```

### Remote scan targets

You can also specify remote targets to scan. The following provides examples of scanning remote targets:

```bash
# to scan a docker image:
cnspec scan docker image ubuntu:22.04

# scan public ECR registry
aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws/r6z5b8t4
cnspec scan docker image public.ecr.aws/r6z5b8t4

# to scan an aws account using the local AWS config
cnspec scan aws

# to scan a kubernetes cluster via your local kubectl config
cnspec scan k8s

# to scan a GitHub repository
export GITHUB_TOKEN=<personal_access_token>
cnspec scan github repo <org/repo> 
```

## Policies

`cnspec` policies are built on the concept of [policy as code](https://mondoo.com/policy-as-code/). `cnspec` comes with default security policies configured for all supported targets. The default policies are available via the [cnspec-policies](https://github.com/mondoohq/cnspec-policies) GitHub repo.

### Custom policies

A `cnspec` policy is simply a YAML file that lets you express any security rule or best practice for your fleet. If you are interested in writing your own policies, or contributing policies back to the `cnspec` community, the best place to start is to look at one of our example policies. You can find them in the [cnspec-policies](https://github.com/mondoohq/cnspec-policies) GitHub repo.

To run a policy you have developed locally:

```bash
cnspec scan local --policy-bundle policy/examples/example.mql.yaml
```

## cnspec Interactive shell

`cnspec` also provides an interactive shell to to explore custom assertions. This will help you understand the queries that are used in policies and write custom queries as well. It’s also a great way to interact with both local and remote targets on the fly.

### Local system shell

```bash
cnspec shell local
 .--. ,-.,-. .--. .---.  .--.  .--.™
'  ..': ,. :`._-.': .; `' '_.''  ..'
`.__.':_;:_;`.__.': ._.'`.__.'`.__.'
   mondoo™        : :
                  :_;
cnquery>
```

#### Using help command

The shell provides a `help` command to get help on the resources that power `cnquery`. Running `help` without any arguments will list all of the available resources and their fields. You can also run `help <resource>` to get more information on a specific resource.

```bash
cnquery> help ports
ports:              TCP/IP ports on the system
  list []port:      TCP/IP ports on the system
  listening []port: TCP/IP ports on the system
```

The shell uses auto-complete which makes it very easy to explore. Once inside the shell, you can enter MQL assertions like this:

```coffeescript
> ports.listening.none( port == 23 )
```

You can type `clear` to clear the terminal. To exit either hit CTRL+D or type `exit`.

## Scaling cnspec across your fleet

The easiest way to scale `cnspec` across your fleet is to have all of your infrastructure pull policies from a central location. The easiest option is to sign-up for a free account on Mondoo Platform. The platform is designed for multi-tenancy, and provides secure private environment that keeps data about your assets in your own account. It makes it very easy for all nodes to report on policies and define custom exceptions for your fleet.

To use `cnquery` with the Mondoo Platform run:

```bash
cnspec auth login
```

Once authenticated, you can scan any target:

```bash
cnspec scan <target>
```

The results from the scan will be returned to `STDOUT` as well as sent back to the platform.

### Uploading policies

To add custom policies to your account, you can now upload policies via:

```bash
cnspec policy upload mypolicy.mql.yaml
```

## Where to go from here

There are so many things cnspec can do! From testing your entire fleet for vulnerabilities to gathering information about it and creating reports for auditors. With its custom policies `cnspec` can scan any component you care about!

Explore our:

- Policy Marketplace
- [Policy as Code](https://mondoo.com/docs/tutorials/mondoo/policy-as-code/)
- [MQL introduction](https://mondoohq.github.io/mql-intro/index.html)
- [MQL resource packs](https://mondoo.com/docs/references/mql/)
- [cnquery](https://github.com/mondoohq/cnquery), our open source, cloud-native asset inventory
- [HashiCorp Packer](https://github.com/mondoohq/packer-plugin-mondoo) - Integrate `cnspec` with HashiCorp Packer!
- Using cnspec with Mondoo


## Community and support

Our goal is to secure all layers of your infrastructure. If you need support, or want to get involved with the development of `cnspec`, join our [community](https://github.com/orgs/mondoohq/discussions) today and let’s grow it together! 

## Troubleshooting

### Private repository access

If you see this error:

```
fatal: could not read Username for 'https://github.com': terminal prompts disabled
Confirm the import path was entered correctly.
```

It is caused by the repository currently being private. You'll have to configure your gitconfig to use SSH to download the repo:

```
[url "ssh://git@github.com/"]
	insteadOf = https://github.com/
```

## Development

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals 📄 docs  
✨⭐🌟🎉 smaller or larger features 🐎 race condition  
🌙 MQL 🌈 visual 🍏 fix tests 🎫 auth 🦅 falcon 🐳 container  

## Legal

- **Copyright:** 2018-2022, Mondoo Inc, proprietary
- **Authors:** Christoph Hartmann, Dominik Richter

