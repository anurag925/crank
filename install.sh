#!/bin/sh
# install.sh — install the `crank` CLI from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/anurag925/crank/main/install.sh | sh
#
# Environment overrides:
#   CRANK_VERSION   Specific version to install (e.g. v0.1.0). Defaults to latest.
#   CRANK_INSTALL_DIR  Target install directory. Defaults to /usr/local/bin
#                    (falls back to $HOME/.local/bin if not writable).
#   CRANK_NO_VERIFY  Set to any value to skip checksum verification.

set -eu

REPO="anurag925/crank"
BINARY="crank"

# --- helpers ---------------------------------------------------------------

info() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

has() { command -v "$1" >/dev/null 2>&1; }

# Download a URL to stdout.
fetch() {
  if has curl; then
    curl -fsSL "$1"
  elif has wget; then
    wget -qO- "$1"
  else
    err "neither curl nor wget is installed"
  fi
}

# Download a URL to a file.
download() {
  # $1 = url, $2 = dest
  if has curl; then
    curl -fsSL -o "$2" "$1"
  elif has wget; then
    wget -qO "$2" "$1"
  else
    err "neither curl nor wget is installed"
  fi
}

# --- detect platform -------------------------------------------------------

detect_os() {
  os="$(uname -s)"
  case "$os" in
    Linux)  echo "linux" ;;
    Darwin) echo "darwin" ;;
    MINGW* | MSYS* | CYGWIN*)
      err "Windows is not supported by this script; download the .zip from https://github.com/$REPO/releases" ;;
    *) err "unsupported OS: $os" ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64 | amd64) echo "amd64" ;;
    aarch64 | arm64) echo "arm64" ;;
    *) err "unsupported architecture: $arch" ;;
  esac
}

# --- resolve version -------------------------------------------------------

resolve_version() {
  if [ -n "${CRANK_VERSION:-}" ]; then
    echo "$CRANK_VERSION"
    return
  fi
  # Resolve the latest release tag from the GitHub API.
  tag="$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name":' \
    | head -n1 \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  [ -n "$tag" ] || err "could not determine latest version; set CRANK_VERSION explicitly"
  echo "$tag"
}

# --- choose install dir ----------------------------------------------------

choose_install_dir() {
  if [ -n "${CRANK_INSTALL_DIR:-}" ]; then
    echo "$CRANK_INSTALL_DIR"
    return
  fi
  if [ -w /usr/local/bin ] 2>/dev/null; then
    echo "/usr/local/bin"
  elif [ "$(id -u)" = "0" ]; then
    echo "/usr/local/bin"
  else
    echo "$HOME/.local/bin"
  fi
}

# --- main ------------------------------------------------------------------

main() {
  OS="$(detect_os)"
  ARCH="$(detect_arch)"
  TAG="$(resolve_version)"
  VERSION="${TAG#v}" # strip leading "v" to match GoReleaser asset names

  ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
  BASE_URL="https://github.com/$REPO/releases/download/$TAG"
  ARCHIVE_URL="$BASE_URL/$ARCHIVE"
  CHECKSUMS_URL="$BASE_URL/checksums.txt"

  info "Installing $BINARY $TAG ($OS/$ARCH)"

  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT INT TERM

  info "Downloading $ARCHIVE"
  download "$ARCHIVE_URL" "$TMP/$ARCHIVE" \
    || err "failed to download $ARCHIVE_URL (does this release/asset exist?)"

  # Verify checksum unless disabled.
  if [ -z "${CRANK_NO_VERIFY:-}" ]; then
    if download "$CHECKSUMS_URL" "$TMP/checksums.txt" 2>/dev/null; then
      info "Verifying checksum"
      expected="$(grep " $ARCHIVE\$" "$TMP/checksums.txt" | awk '{print $1}')"
      if [ -n "$expected" ]; then
        if has sha256sum; then
          actual="$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')"
        elif has shasum; then
          actual="$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')"
        else
          actual=""
          warn "no sha256 tool found; skipping verification"
        fi
        if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
          err "checksum mismatch (expected $expected, got $actual)"
        fi
      else
        warn "no checksum entry for $ARCHIVE; skipping verification"
      fi
    else
      warn "could not download checksums.txt; skipping verification"
    fi
  fi

  info "Extracting"
  tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
  [ -f "$TMP/$BINARY" ] || err "binary '$BINARY' not found in archive"

  DIR="$(choose_install_dir)"
  mkdir -p "$DIR"

  if [ -w "$DIR" ]; then
    install -m 0755 "$TMP/$BINARY" "$DIR/$BINARY"
  elif has sudo; then
    warn "$DIR is not writable; using sudo"
    sudo install -m 0755 "$TMP/$BINARY" "$DIR/$BINARY"
  else
    err "$DIR is not writable and sudo is unavailable; set CRANK_INSTALL_DIR to a writable path"
  fi

  info "Installed $BINARY to $DIR/$BINARY"

  case ":$PATH:" in
    *":$DIR:"*) ;;
    *) warn "$DIR is not on your PATH; add it, e.g.: export PATH=\"$DIR:\$PATH\"" ;;
  esac

  if has "$BINARY"; then
    info "Done: $("$BINARY" --version 2>/dev/null || echo "$BINARY installed")"
  else
    info "Done. Run: $DIR/$BINARY --version"
  fi
}

main "$@"
