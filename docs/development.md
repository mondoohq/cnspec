# Development

## Build

### Prerequisites

Before building from source, be sure to install:

- [Go 1.22.0+](https://go.dev/dl/)
- [Protocol Buffers v29+](https://github.com/protocolbuffers/protobuf/releases)

On macOS systems with Homebrew, run: `brew install go@1.22 protobuf`

## Install from source

1. Verify that you have Go 1.22+ installed:

    ```
    $ go version
    ```

If `go` is not installed or an older version exists, follow instructions on [the Go website](https://go.dev/doc/install).

2. Clone this repository:

   ```sh
   $ git clone https://github.com/mondoohq/cnspec.git
   $ cd cnspec
   ```

3. Build and install:

    #### Unix-like systems
    ```sh
    # To install cnspec using Go into the $GOBIN directory:
    make cnspec/install
    ```

## Develop cnspec

Whenever you change protos or other auto-generated files, you must regenerate files for the compiler. To do this, be sure you have the necessary tools installed (such as protobuf):

```bash
make prep
```

You also need to have the required dependencies present:

```bash
make prep/repos
```

When the repo is already present and something changed upstream, update the dependencies:

```bash
make prep/repos/update
```

Then, whenever you make changes, just run:

```bash
make cnspec/generate
```

This generates and updates all required files for the build. At this point you can `make cnspec/install` again as outlined above.

## Contribute changes

### Mark PRs with emojis

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals ⚡ speed 📄 docs  
✨⭐🌟🌠 smaller or larger features 🐎 race condition  
🌙 MQL 🌈 visual 🟢 fix tests 🎫 auth 🦅 falcon 🐳 container  

