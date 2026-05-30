---
description: Build, release, and publish to Homebrew
argument-hint: <patch|minor|major>
---

# Release

Perform a full release of the `agent-stripe` CLI: version bump, tag, build,
GitHub release, and Homebrew tap update.

## Arguments

- `$ARGUMENTS` - version bump type: `patch`, `minor`, or `major`

## Instructions

You are performing a release of the `agent-stripe` CLI (Go version). Follow
these steps exactly.

### Pre-flight

1. Confirm `$ARGUMENTS` is exactly `patch`, `minor`, or `major`. If not, stop and ask.
2. Confirm the working tree is clean:
   ```bash
   git status --short
   ```
   If there are changes, stop and ask.
3. Confirm the current branch is `main` and it is up to date with `origin/main`.
4. Run:
   ```bash
   make test
   go vet ./...
   ```
   If either fails, stop and fix.
5. Determine the current version from the latest git tag:
   ```bash
   current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
   ```
   If no release tag exists, treat the first release as starting from `0.1.0`
   for a `minor` bump or `0.0.1` for a `patch` bump.

### Step 1: Version bump, tag, and push

Calculate the next version:

```bash
current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
IFS='.' read -r major minor patch <<< "$current"

case "$ARGUMENTS" in
  patch)
    patch=$((patch + 1))
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  *)
    echo "expected patch, minor, or major"
    exit 1
    ;;
esac

new_version="${major}.${minor}.${patch}"
echo "Releasing v${new_version}"
```

Then tag and push:

```bash
git tag "v${new_version}"
git push origin main "v${new_version}"
```

### Step 2: Build with GoReleaser

Preferred path:

```bash
goreleaser release --clean
```

If `goreleaser` is not installed, build manually:

```bash
rm -rf dist/
mkdir -p dist
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-darwin-arm64" ./cmd/agent-stripe
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-darwin-amd64" ./cmd/agent-stripe
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-linux-amd64" ./cmd/agent-stripe
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-linux-arm64" ./cmd/agent-stripe
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=${new_version}" -o "dist/agent-stripe-windows-amd64.exe" ./cmd/agent-stripe

cd dist
for bin in agent-stripe-darwin-arm64 agent-stripe-darwin-amd64 agent-stripe-linux-amd64 agent-stripe-linux-arm64; do
  tar czf "${bin}.tar.gz" "$bin"
done
zip agent-stripe-windows-amd64.zip agent-stripe-windows-amd64.exe
shasum -a 256 *.tar.gz *.zip > checksums-sha256.txt
cd ..
```

Smoke-test the native binary before proceeding:

```bash
./dist/agent-stripe-darwin-arm64 --version
./dist/agent-stripe-darwin-arm64 usage
```

### Step 3: Create GitHub release

If GoReleaser created the GitHub release, verify it and skip the manual create:

```bash
gh release view "v${new_version}"
```

Otherwise:

```bash
prev_tag=$(git tag --sort=-v:refname | head -2 | tail -1)
notes=$(git log --pretty=format:"- %s" "${prev_tag}..v${new_version}" --no-merges | grep -v "^- v[0-9]" || true)

gh release create "v${new_version}" dist/*.tar.gz dist/*.zip dist/checksums-sha256.txt \
  --title "v${new_version}" \
  --notes "$notes"
```

Verify:

```bash
gh release view "v${new_version}"
```

### Step 4: Update Homebrew tap

The Homebrew formula lives in `../homebrew-tap` relative to this repo root.

```bash
ls ../homebrew-tap/Formula/agent-stripe.rb
```

If it does not exist, create it by copying the pattern from
`../homebrew-tap/Formula/agent-sql.rb`, replacing:

- Class name: `AgentStripe`
- desc: `"Stripe incident triage CLI for AI agents"`
- homepage: `https://github.com/shhac/agent-stripe`
- all `agent-sql` references with `agent-stripe`
- version, URLs, and SHA256 values
- test assertions for `agent-stripe --version` and `agent-stripe usage`

If it exists, read checksums from `dist/checksums-sha256.txt` and update:

1. `../homebrew-tap/Formula/agent-stripe.rb`
2. Version and release URLs using `v${new_version}`
3. SHA256 values for darwin/linux arm64/amd64 archives
4. Formula test version assertion

Then commit and push the tap:

```bash
cd ../homebrew-tap
git status --short
git add Formula/agent-stripe.rb
git commit -m "agent-stripe ${new_version}"
git push
cd -
```

Always return to the `agent-stripe` repo after updating the tap.

### Step 5: Report

Show the user:

- New version number
- GitHub release URL
- Homebrew tap commit, if applicable
- `brew install shhac/tap/agent-stripe`
- `brew upgrade shhac/tap/agent-stripe`
