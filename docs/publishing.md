# Publishing

How a plankit release becomes binaries, a Homebrew formula, and an update notice. The tag is the version everywhere; `pk release` creates and pushes it, and everything below reacts to it.

## From tag to release

`.github/workflows/release.yml` runs on a `v*` tag push:

1. Builds `pk` for five targets (darwin and linux on amd64 and arm64, windows on amd64), `CGO_ENABLED=0`, the version injected by ldflags from the tag.
2. Writes `checksums.txt` (`sha256sum` over the five binaries).
3. Creates the GitHub Release with generated notes, the binaries, and the checksums.
4. Notifies the Homebrew tap (`markwharton/homebrew-plankit`) with a `repository_dispatch`; the step is fire-and-forget, and the tap's daily schedule covers a failed ping.

## How consumers get it

- **Homebrew**: `brew tap markwharton/plankit && brew install plankit`; the tap's formula follows the latest release.
- **Go**: `go install github.com/markwharton/plankit/cmd/pk@latest`.
- **Binary**: the [releases page](https://github.com/markwharton/plankit/releases); `.claude/install-pk.sh` downloads the same artifacts, pinned to a version and checksum-verified, in cloud sandboxes.

## How consumers hear about it

`pk version` checks the latest release daily and prints the upgrade command that matches the install (`brew upgrade plankit` for Homebrew, `go install …@latest` otherwise). `pk setup` pins the running version into `.claude/install-pk.sh`; `pk version` notes when pin and binary differ ([pk-version.md](pk-version.md)).

## Limits

- The release workflow needs the tag pushed by `pk release`; a hand-pushed tag triggers it identically, which is why the branch model routes tags through `pk release` only.
- The formula update is asynchronous: between the release and the tap's bump, `brew install` serves the previous version.
- No package registries beyond Homebrew and the Go module proxy; no signed binaries or SLSA provenance today. Reopen with a second maintainer or a request from a consumer that verifies signatures.
