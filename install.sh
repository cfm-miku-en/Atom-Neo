#!/bin/sh
# Installs the atom command on Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/cfm-miku-en/Atom-Neo/main/install.sh | sh
#
# Override with environment variables if you want to:
#   ATOM_VERSION=v0.1.0            pin a release instead of taking the latest
#   ATOM_INSTALL_DIR=/opt/atom     where the files go
#   ATOM_BIN_DIR=/usr/local/bin    where the atom command goes
#
# Windows has its own installer; grab AtomNeoSetup from the releases page.

set -eu

REPO="cfm-miku-en/Atom-Neo"
INSTALL_DIR="${ATOM_INSTALL_DIR:-$HOME/.local/share/atom-neo}"
BIN_DIR="${ATOM_BIN_DIR:-$HOME/.local/bin}"

say() { printf '  %s\n' "$1"; }

die() {
	printf '\nFailed: %s\n' "$1" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required but not installed"
}

printf '\nAtom-Neo\n\n'

case "$(uname -s)" in
	Linux) ;;
	Darwin) die "macOS is not supported. Build from source: go build -o atom ./src" ;;
	*) die "unsupported system: $(uname -s)" ;;
esac

case "$(uname -m)" in
	x86_64 | amd64) ARCH="amd64" ;;
	aarch64 | arm64) ARCH="arm64" ;;
	*) die "unsupported architecture: $(uname -m)" ;;
esac

need curl
need tar
need sha256sum

VERSION="${ATOM_VERSION:-}"
if [ -z "$VERSION" ]; then
	say "looking up the latest release..."
	VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
		grep '"tag_name"' | head -1 | cut -d'"' -f4)

	# A release that is still a draft is invisible here, which is the usual
	# reason for coming up empty.
	[ -n "$VERSION" ] || die "no published release found. Pass ATOM_VERSION=v0.1.0 to pick one."
fi

ARCHIVE="atom-neo-linux-$ARCH.tar.gz"
BASE="https://github.com/$REPO/releases/download/$VERSION"

say "installing $VERSION for linux/$ARCH"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$BASE/$ARCHIVE" -o "$TMP/$ARCHIVE" ||
	die "could not download $ARCHIVE for $VERSION"

# The release publishes checksums, so use them rather than trusting the
# download arrived intact.
if curl -fsSL "$BASE/SHA256SUMS.txt" -o "$TMP/SHA256SUMS.txt" 2>/dev/null; then
	# sha256sum writes "hash filename" in text mode and "hash *filename" in
	# binary mode; matching only one of those would skip the check silently.
	expected=$(awk -v want="$ARCHIVE" '$2 == want || $2 == "*" want { print $1 }' "$TMP/SHA256SUMS.txt")
	actual=$(sha256sum "$TMP/$ARCHIVE" | cut -d' ' -f1)

	if [ -n "$expected" ] && [ "$expected" != "$actual" ]; then
		die "checksum mismatch for $ARCHIVE. Not installing."
	fi
	say "checksum verified"
else
	say "no checksums published for this release, skipping verification"
fi

mkdir -p "$INSTALL_DIR" "$BIN_DIR"
tar -xzf "$TMP/$ARCHIVE" -C "$INSTALL_DIR"

[ -f "$INSTALL_DIR/atom" ] || die "the archive did not contain an atom binary"
chmod +x "$INSTALL_DIR/atom"

ln -sf "$INSTALL_DIR/atom" "$BIN_DIR/atom"
say "installed to $BIN_DIR/atom"

printf '\n'
case ":$PATH:" in
	*":$BIN_DIR:"*)
		printf 'Done. Try:\n    atom repl\n\n'
		;;
	*)
		# Adding this silently to a shell profile would be a surprise, so it is
		# printed instead.
		printf 'Done, but %s is not on your PATH.\n' "$BIN_DIR"
		printf 'Add this to your shell profile:\n\n'
		printf '    export PATH="%s:$PATH"\n\n' "$BIN_DIR"
		;;
esac
