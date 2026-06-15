# dsh-go

A Go implementation of [dsh](https://www.netfort.gr.jp/~dancer/software/dsh.html.en) (Distributed Shell) — run a command on multiple hosts over SSH simultaneously.

## Install

**With Go:**
```sh
go install github.com/goldbach/dsh-go/cmd/dsh@latest
```

**From a release tarball:**

Download the archive for your platform from the [releases page](https://github.com/goldbach/dsh-go/releases), extract it, and copy the `dsh` binary to somewhere on your `PATH`:

```sh
tar -xzf dsh-go_linux_amd64.tar.gz
sudo mv dsh /usr/local/bin/
```

## Usage

```
dsh [options] [--] <command> [args...]
```

### Host selection

| Flag | Description |
|------|-------------|
| `-g <group>` | Group from `~/.dsh/group/<name>` (repeatable) |
| `-m <host>` | Individual host (repeatable) |
| `-f <file>` | File containing host names, one per line (repeatable) |
| `-a` | All hosts from `~/.dsh/machines.list` |

Flags may be combined; duplicates are silently deduplicated.

### Execution

| Flag | Description |
|------|-------------|
| `-c` | Concurrent — run on all hosts at once (default) |
| `-w` | Sequential — wait for each host before moving to the next |
| `-F <n>` | Max concurrent SSH connections (default: 64) |

### Output

| Flag | Description |
|------|-------------|
| `-H` | Hide hostname prefix (shown by default) |

### SSH

| Flag | Description |
|------|-------------|
| `-o <arg>` | Extra argument passed to `ssh` before the hostname (repeatable) |

## Configuration

Group files live in `~/.dsh/group/`. Each file contains one host per line; `#` starts a comment.

```
# ~/.dsh/group/web
web1.example.com
web2.example.com
# web3.example.com  (disabled)
```

`~/.dsh/machines.list` is the default pool used by `-a`, in the same format.

Hosts may be written as `user@host` to override the SSH username per entry.

## Examples

```sh
# Run uptime on all hosts in the "web" group
dsh -g web uptime

# Run df across two groups concurrently, show which host each line came from
dsh -g web -g db df -h

# Sequential run on ad-hoc hosts
dsh -w -m host1 -m host2 -- sudo apt upgrade -y

# Pass a custom SSH port
dsh -g staging -o "-p 2222" uptime

# Use all hosts from machines.list, cap to 10 concurrent connections
dsh -a -F 10 -- systemctl restart myapp
```

## Differences from original dsh

- SSH only — `rsh`/`remsh` are not supported.
- Machine names are shown by default; use `-H` to suppress.
- No `dsh.conf` global config file.
