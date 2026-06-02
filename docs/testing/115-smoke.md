# 115 Smoke Test

Use an isolated config directory:

```bash
export PAN_CLI_CONFIG_DIR=/tmp/pan-cli-115-smoke
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
go run ./cmd/115-cli --json mkdir / pan-cli-smoke
go run ./cmd/115-cli --json rename /pan-cli-smoke pan-cli-smoke-renamed
go run ./cmd/115-cli --json rm /pan-cli-smoke-renamed
```
