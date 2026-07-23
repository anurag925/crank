---
title: Installation
---

# Installation

This page covers installing the `crank` CLI. Generated projects have their own dependencies, which `crank init` installs automatically with `go get` and `go mod tidy`.

## Requirements

Install these first:

- Go 1.21 or newer.
- Git.
- Network access when scaffolding projects, because dependencies are fetched with `go get`.

Optional tools such as `migrate`, `swag`, and `air` are installed automatically when possible.

## Install with the install script

Recommended for most users:

```bash
curl -fsSL https://raw.githubusercontent.com/anurag925/crank/main/install.sh | sh
```

The script detects your operating system and architecture, downloads the matching release, verifies checksums, and installs the `crank` binary.

Useful environment variables:

| Variable | Purpose |
| --- | --- |
| `CRANK_VERSION` | Install a specific release, for example `v0.1.0`. |
| `CRANK_INSTALL_DIR` | Choose the destination directory. |
| `CRANK_NO_VERIFY` | Skip checksum verification. |

Example:

```bash
CRANK_VERSION=v0.1.0 CRANK_INSTALL_DIR="$HOME/bin" \
  sh -c "$(curl -fsSL https://raw.githubusercontent.com/anurag925/crank/main/install.sh)"
```

Make sure the install directory is on your `PATH`:

```bash
crank --version
```

## Install with Go

If you already have Go configured:

```bash
go install github.com/anurag925/crank/cmd/crank@latest
```

Then verify:

```bash
crank --version
```

If your shell cannot find `crank`, add Go's binary directory to your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Build from source

Clone the repository and build the binary:

```bash
git clone https://github.com/anurag925/crank.git
cd crank
make build
./bin/crank --help
```

You can also install the local checkout into your Go binary directory:

```bash
make install
```

## Verify the installation

Run:

```bash
crank --help
crank list
crank tools
```

You should see the root help, the feature list, and the tool subcommand list.

## Installing tool wrappers

`crank` wraps external tools, but you usually do not need to install them manually.

| Tool command | External binary | Auto-installed? |
| --- | --- | --- |
| `crank migrate` | `migrate` | Yes, via `go install` with PostgreSQL support. |
| `crank make swag` | `swag` | Yes. |
| `crank dev` | `air` | Yes. |
| `crank build` | `go` | Comes from your Go installation. |
| `crank run` | `go` | Comes from your Go installation. |
| `crank test` | `go` | Comes from your Go installation. |
| `crank gofmt` | `gofmt` | Comes from your Go installation. |
| `crank vet` | `go` | Comes from your Go installation. |
| `crank tidy` | `go` | Comes from your Go installation. |
| `crank doctor` | none | Runs in-process. |

If automatic installation fails, run the command shown by `crank` manually and ensure the binary directory is on your `PATH`.
