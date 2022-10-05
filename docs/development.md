### Dependencies

## Prerequisites

Before building from source, be sure to install:

- [Go 1.19.0+](https://golang.org/dl/)
- [Protocol Buffers v21+](https://github.com/protocolbuffers/protobuf/releases)

On macOS systems with Homebrew, run: `brew install go@1.19 protobuf`

## Installation from source

1. Verify that you have Go 1.19+ installed

    ```
    $ go version
    ```

If `go` is not installed or an older version exists, follow instructions on [the Go website](https://golang.org/doc/install).

2. Clone this repository

   ```sh
   $ git clone https://github.com/mondoohq/cnspec.git
   $ cd cnspec
   ```

3. Build and install

   #### Unix-like systems
    ```sh
    # To install `cnspec` using Go into the $GOBIN directory:
    export GOPRIVATE="github.com/mondoohq,go.mondoo.com"
    make cnspec/install
    ```

## Develop cnspec

Whenever you change protos or other auto-generated files, you must regenerate files for the compiler. To do this, be sure you have the necessary tools installed (such as protobuf):

```bash
make prep
```

Then, whenever you make changes, just run:

```bash
make cnspec/generate
```

This generates and updates all required files for the build. Now you can `make cnspec/install` again as outlined above.
