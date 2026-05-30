---
description: Build, release, and publish to Homebrew
argument-hint: <patch|minor|major>
---

# Release

Perform a full release: version bump, build, GitHub release, and Homebrew tap update.

## Arguments

- `$ARGUMENTS` — version bump type: `patch`, `minor`, or `major`

## Instructions

You are performing a release of the `agent-stripe` CLI (Go version). Follow these steps exactly.

### Pre-flight

1. Confirm the working tree is clean (`git status --short`). If not, stop and ask.
2. Run `make test` and `go vet ./...`. If either fails, stop and fix.
3. Determine the current version from the latest git tag (`git describe --tags --abbrev=0`) and show what bump will happen. If no tag exists, start at `0.1.0`.

### Step 1: Version bump, tag, and push

Calculate the new version by bumping the current tag:

```bash
current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
IFS='.' read -r major minor patch <<< "$current"
```

Apply the bump type:

- `patch`: increment patch
- `minor`: increment minor, reset patch to 0
- `major`: increment major, reset minor and patch to 0

Then tag and push:

```bash
git tag "v${new_version}"
git push origin main "v${new_version}"
```

### Step 2: Build with goreleaser

```bash
goreleaser release --clean
```

If `goreleaser` is not installed, build manually:

```bash
rm -rf dist/
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-darwin-arm64" ./cmd/agent-stripe
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-darwin-amd64" ./cmd/agent-stripe
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-linux-amd64" ./cmd/agent-stripe
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-linux-arm64" ./cmd/agent-stripe
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-windows-amd64.exe" ./cmd/agent-stripe
cd dist
for bin in agent-stripe-darwin-arm64 agent-stripe-darwin-amd64 agent-stripe-linux-amd64 agent-stripe-linux-arm64; do
  tar czf "${bin}.tar.gz" "$bin"
done
shasum -a 256 *.tar.gz agent-stripe-windows-amd64.exe > checksums-sha256.txt
cd ..
```

Smoke-test the native binary:

```bash
./dist/agent-stripe-darwin-arm64 --version
./dist/agent-stripe-darwin-arm64 usage
```

### Step 3: Create GitHub release

If goreleaser handled it, skip this step. Otherwise:

```bash
prev_tag=$(git tag --sort=-v:refname | head -2 | tail -1)
notes=$(git log --pretty=format:"- %s" "${prev_tag}..v${new_version}" --no-merges | grep -v "^- v[0-9]")
gh release create "v${new_version}" dist/*.tar.gz dist/agent-stripe-windows-amd64.exe dist/checksums-sha256.txt \
  --title "v${new_version}" \
  --notes "$notes"
```

### Step 4: Update Homebrew tap

The Homebrew formula lives in `../homebrew-tap` relative to this repo's root.

Copy the pattern from sibling `agent-*` formulae and update:

- Class name: `AgentStripe`
- desc: `"Stripe incident triage CLI for AI agents"`
- homepage: `https://github.com/shhac/agent-stripe`
- Version, URLs, and SHA256 values
- Test: assert version and usage output

### Step 5: Report

Show the user:

- New version number
- GitHub release URL
- Homebrew tap commit, if applicable
- `brew install shhac/tap/agent-stripe`
- `brew upgrade shhac/tap/agent-stripe`
