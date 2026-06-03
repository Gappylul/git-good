# git-good

A lightweight CLI tool that installs a Git pre-commit hook to scan staged changes for hardcoded secrets and credentials, using rules you define in a simple YAML config.

---

## How it works

1. Run `git-good init` to generate a `git-good.yaml` config in your repo root.
2. Edit the config to add your own regex rules, exclusions, and an optional webhook.
3. Run `git-good apply` to validate the config and install a `pre-commit` hook in `.git/hooks/`.
4. From that point on, every `git commit` automatically scans staged changes. If a rule matches, the commit is blocked.

---

## Installation

### Download a release

Grab the latest binary for your platform from the [Releases](https://github.com/Gappylul/git-good/releases) page and place it somewhere on your `$PATH`.

### Build from source

```bash
git clone https://github.com/Gappylul/git-good
cd git-good
go build -o git-good ./cmd/
```

Or with metadata injected:

```bash
task build
```

---

## Usage

```
git-good [options] <command>

Commands:
  init       Generate a starter git-good.yaml in the current directory
  apply      Validate git-good.yaml and install the pre-commit hook
  run-hook   Scan staged changes for secrets (invoked automatically by the hook)

Options:
  -v         Print short version string
  -version   Print detailed build info (version, commit, build time)
```

---

## Configuration

`git-good.yaml` lives in the root of your repository.

```yaml
version: "1"

settings:
  fail_on_match: true          # Block the commit when a rule matches
  webhook_url: "https://..."   # Optional: POST match details to a webhook

exclude:
  - "*.png"
  - "node_modules/*"
  - "go.sum"

rules:
  - name: AWS Access Key
    pattern: "AKIA[0-9A-Z]{16}"

  - name: Generic Secret String
    pattern: '(?i)(password|secret|api_key|passwd)\s*=\s*[''"][^''"]+[''"]'
```

Each rule needs a `name` (shown in the output when triggered) and a `pattern` (a Go-compatible regular expression). Patterns are matched only against added lines in the diff - deletions are ignored.

When `fail_on_match` is `true`, a matching commit is blocked with a non-zero exit code. Set it to `false` to warn without blocking.

---

## Example output

```
[git-good] COMMIT BLOCKED!
It looks like you're trying to commit credentials or secrets:
------------------------------------------------------------
  File            : config.env
  Rule Triggered  : AWS Access Key
  Leaked String   : AWS_KEY=AKIAIOSFODNN7EXAMPLE
------------------------------------------------------------
💡 Remove the secrets from your code, stage the changes, and try again.
```

---

## Webhook alerts

Set `webhook_url` in your config and git-good will POST a JSON payload to that URL whenever a match is detected - before blocking the commit. The request times out after 2 seconds and a failed dispatch is logged to stderr but never blocks or overrides `fail_on_match`.

Payload shape:

```json
{
  "event": "secret_detected",
  "timestamp": "2026-06-03T10:00:00Z",
  "repo_name": "",
  "matches": [
    {
      "RuleName": "AWS Access Key",
      "FileName": "config.env",
      "Content": "AWS_KEY=AKIAIOSFODNN7EXAMPLE"
    }
  ]
}
```

Any HTTP server that accepts a POST with a JSON body works. This includes Discord and Slack incoming webhooks - just paste their webhook URL directly into `webhook_url` and note that they expect a `content` or `text` field, so you'll need a small proxy or middleware to reformat the payload if you want rich formatting. For a plain-text ping, a free service like [webhook.site](https://webhook.site) is the easiest way to route alerts into any channel.

---

## Development

```bash
# Run tests with race detector
task test

# Build production binary into dist/
task build

# Remove build artifacts
task clean
```

Releases are built with [GoReleaser](https://goreleaser.com) for Linux, macOS, and Windows across `amd64` and `arm64`.

---

## License

MIT - see [LICENSE](LICENSE) for details.