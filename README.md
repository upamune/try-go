# try-go

`try-go` is a Go implementation of [`tobi/try`](https://github.com/tobi/try).

## Install

### Homebrew

```bash
brew install upamune/tap/try
```

### Go install

```bash
go install github.com/upamune/try-go/cmd/try@latest
```

### mise

```bash
mise use -g go:github.com/upamune/try-go/cmd/try@latest
```

After installing `try`, enable shell integration:

```bash
eval "$(try init)"
```

## Usage

```bash
# interactive selector
try exec

# shorthand (same as exec)
try my-project

# clone workflow
try clone https://github.com/user/repo
```

## Development

```bash
mise trust
mise install
mise run build
```
