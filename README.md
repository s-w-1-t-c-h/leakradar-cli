# leakradar-cli

[![CI](https://github.com/s-w-1-t-c-h/leakradar-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/s-w-1-t-c-h/leakradar-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/s-w-1-t-c-h/leakradar-cli)](https://goreportcard.com/report/github.com/s-w-1-t-c-h/leakradar-cli)
[![Latest release](https://img.shields.io/github/v/release/s-w-1-t-c-h/leakradar-cli)](https://github.com/s-w-1-t-c-h/leakradar-cli/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/s-w-1-t-c-h/leakradar-cli)](go.mod)
[![License](https://img.shields.io/github/license/s-w-1-t-c-h/leakradar-cli)](LICENSE)

A cross-platform (Linux/macOS/Windows) command-line client for the
[LeakRadar](https://leakradar.io) breach/leak-intelligence API
(`https://api.leakradar.io`): search by email, domain, or advanced
multi-field filters, check dark-web mentions, batch-check large lists,
unlock and export results, all scriptable via `--json`.

## Install

**Prebuilt binary**: download the archive for your OS/arch from the
[Releases page](https://github.com/s-w-1-t-c-h/leakradar-cli/releases),
extract, and put `leakradar-cli` (or `leakradar-cli.exe`) on your `PATH`.

**From source** (requires Go 1.26+):

```sh
go install ./cmd/leakradar-cli
```

or from within this directory:

```sh
go build -o leakradar-cli ./cmd/leakradar-cli
```

## Authentication

Your API key is never hardcoded or committed. It's resolved in this order:

1. `--api-key` flag (try not to do this as your key will end up in your shell history)
2. `LEAKRADAR_API_KEY` environment variable
3. OS keychain (macOS Keychain / Windows Credential Manager / Linux Secret Service)
4. `config.json` in the OS config dir (`~/.config/leakradar-cli/`, `~/Library/Application Support/leakradar-cli/`,
   or `%AppData%\leakradar-cli\`), written with `0600` permissions as a fallback when no OS keychain is available

Set it up:

```sh
leakradar-cli auth set          # prompts for the key with masked input
leakradar-cli auth status       # validates the key against /profile
leakradar-cli auth clear        # removes it from keychain + config file
```

## Global flags

- `--json`: machine-readable output on every command, for piping to `jq`
- `--outdir <dir>`: save every result to an organised directory tree (see below)
- `--timeout`: per-request HTTP timeout (default 30s)
- `-v/--verbose`: verbose logging (never logs the key itself)
- `--base-url`: override the API base URL (testing only)

The client self-throttles to stay under LeakRadar's documented rate limits
(30 req/s edge, 5 req/s for `/search/advanced`) and automatically retries
`429`/`503` responses with backoff, honouring `Retry-After`.

## Saving results to disk (`--outdir`)

Every command can additionally write its result to disk instead of, or
alongside, stdout: a `.json` file plus a `.txt` with the same table shown
on screen, organised by target (domain/email/hash), plus a running
`audit.jsonl` log:

```sh
leakradar-cli --outdir ./engagement domain example.com
# or set it once for the shell session:
export LEAKRADAR_OUTDIR=./engagement
leakradar-cli domain example.com
```

Saving is best-effort: a write problem warns on stderr rather than
aborting the command.

## Commands

Run `leakradar-cli examples` for a full, runnable example of every command
(`leakradar-cli examples <keyword>` to filter, e.g. `raw`/`unlock`/`combolist`),
or `--help` on any command for its full flag list.

```sh
leakradar-cli profile
leakradar-cli balance
leakradar-cli email jsmith@example.com
leakradar-cli domain example.com
leakradar-cli advanced --url-domain example.com --password-strength weak
leakradar-cli darkweb --query "example.com"
leakradar-cli password-range 5BAA6
leakradar-cli batch emails emails.txt
leakradar-cli unlock email jsmith@example.com
leakradar-cli export email jsmith@example.com --format csv
leakradar-cli combolist email jsmith@example.com
leakradar-cli raw search --q example.com
leakradar-cli version
```

Every list/search command accepts `--json` for scripting:

```sh
leakradar-cli email jsmith@example.com --json | jq '.items[].url'
```

## License

[MIT](LICENSE)
