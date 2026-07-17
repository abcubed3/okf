#!/bin/sh
# OKF-go CLI Installer
# Usage: curl -sSfL https://raw.githubusercontent.com/abcubed3/okf/main/install.sh | sh
# Or specify version: curl -sSfL https://raw.githubusercontent.com/abcubed3/okf/main/install.sh | sh -s -- v1.0.0

set -e

OWNER="abcubed3"
REPO="okf"
BINARY="okf"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
  darwin)  OS="darwin" ;;
  linux)   OS="linux" ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *)       echo "Unsupported OS: ${OS}. Only macOS, Linux, and Windows are supported by this script."; exit 1 ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64)
    if [ "${OS}" = "windows" ]; then
      echo "Unsupported architecture: Windows ARM64 builds are not available."
      exit 1
    fi
    ARCH="arm64"
    ;;
  *)            echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

# Set binary name extension and archive format
if [ "${OS}" = "windows" ]; then
  BINARY_EXE="${BINARY}.exe"
  EXT="zip"
else
  BINARY_EXE="${BINARY}"
  EXT="tar.gz"
fi

# Determine version
TAG="$1"
if [ -z "${TAG}" ]; then
  echo "Checking for the latest release version..."
  # Try GitHub Redirect location headers first (very fast, no rate limit issues)
  TAG=$(curl -sI "https://github.com/${OWNER}/${REPO}/releases/latest" | grep -Ei '^location:' | grep -oE 'tag/v[0-9.]+' | cut -d/ -f2 || true)
  if [ -z "${TAG}" ]; then
    # Fallback to the GitHub API
    TAG=$(curl -s "https://api.github.com/repos/${OWNER}/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
  fi
fi

if [ -z "${TAG}" ] || [ "${TAG}" = "null" ]; then
  echo "Error: Could not retrieve the latest version tag."
  echo "You can specify a version tag manually, e.g.:"
  echo "  curl -sSfL https://raw.githubusercontent.com/${OWNER}/${REPO}/main/install.sh | sh -s -- v1.0.0"
  exit 1
fi

# Clean version name (strip leading 'v' for GoReleaser archive name)
VERSION="${TAG#v}"

echo "Selected release tag: ${TAG}"
echo "Downloading OKF CLI ${TAG} for ${OS}/${ARCH}..."

TARBALL="${BINARY}_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${OWNER}/${REPO}/releases/download/${TAG}/${TARBALL}"

# Create temporary directory
TMP_DIR=$(mktemp -d)
clean_up() {
  rm -rf "${TMP_DIR}"
}
trap clean_up EXIT INT TERM

# Download archive
if command -v curl >/dev/null 2>&1; then
  curl -sSfL -o "${TMP_DIR}/${TARBALL}" "${URL}"
elif command -v wget >/dev/null 2>&1; then
  wget -q -O "${TMP_DIR}/${TARBALL}" "${URL}"
else
  echo "Error: Neither curl nor wget was found on the system."
  exit 1
fi

# Extract binary
tar -xf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}" "${BINARY_EXE}"

# Determine installation directory
# Order of preference:
# 1. $INSTALL_DIR env variable if set
# 2. /usr/local/bin if writeable (non-Windows only)
# 3. ~/.local/bin if writeable or created
# 4. ./bin in current directory
if [ -n "${INSTALL_DIR}" ]; then
  TARGET_DIR="${INSTALL_DIR}"
elif [ "${OS}" != "windows" ] && [ -w "/usr/local/bin" ]; then
  TARGET_DIR="/usr/local/bin"
elif [ -d "${HOME}/.local/bin" ] && [ -w "${HOME}/.local/bin" ]; then
  TARGET_DIR="${HOME}/.local/bin"
else
  # Check if we can create ~/.local/bin
  mkdir -p "${HOME}/.local/bin" >/dev/null 2>&1 || true
  if [ -w "${HOME}/.local/bin" ]; then
    TARGET_DIR="${HOME}/.local/bin"
  else
    TARGET_DIR="./bin"
    mkdir -p "${TARGET_DIR}"
  fi
fi

# Move binary to target directory
mv "${TMP_DIR}/${BINARY_EXE}" "${TARGET_DIR}/${BINARY_EXE}"
chmod +x "${TARGET_DIR}/${BINARY_EXE}"

echo "Successfully installed ${BINARY} to ${TARGET_DIR}/${BINARY_EXE}"

# Output PATH instructions if not in /usr/local/bin
if [ "${TARGET_DIR}" != "/usr/local/bin" ]; then
  case ":${PATH}:" in
    *:"${TARGET_DIR}":*) ;;
    *)
      # Resolve absolute path for instructions
      ABS_TARGET_DIR=$(cd "${TARGET_DIR}" && pwd)
      echo ""
      echo "⚠️  Important: Make sure to add the installation directory to your PATH:"
      echo "  export PATH=\"\${PATH}:${ABS_TARGET_DIR}\""
      echo "Add the above line to your ~/.bashrc, ~/.zshrc, or profile file to make it permanent."
      if [ "${OS}" = "windows" ]; then
        echo "Or add the Windows path to your System Environment Variables:"
        echo "  ${ABS_TARGET_DIR}"
      fi
      ;;
  esac
fi
