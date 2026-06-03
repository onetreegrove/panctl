# 115 Smoke Test

Use an isolated config directory:

```bash
export PANCTL_CONFIG_DIR=/tmp/panctl-115-smoke
```

Login:

```bash
printf '%s' "$PAN_115_COOKIE" | go run ./cmd/115-cli --json login cookie --stdin
```

Read-only checks:

```bash
go run ./cmd/115-cli --json login status
go run ./cmd/115-cli --json ls /
go run ./cmd/115-cli --json download /path/to/small-file
```

Mutation checks in a disposable directory:

```bash
go run ./cmd/115-cli --json mkdir / panctl-smoke
go run ./cmd/115-cli --json rename /panctl-smoke panctl-smoke-renamed
go run ./cmd/115-cli --json rm /panctl-smoke-renamed
```
