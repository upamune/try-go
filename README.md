# try-go

`try-go` is a Go implementation of [`tobi/try`](https://github.com/tobi/try).

## Install

### Go install

```bash
go install github.com/upamune/try-go/cmd/try@latest
```

### mise

```bash
mise use -g go@1.24
mise use -g pinact@3.4.4
mise install
```

Then build/install from source if needed:

```bash
go build -o try ./cmd/try
```

## Usage

```bash
# interactive selector
try exec

# shorthand (same as exec)
try my-project

# clone workflow
try clone https://github.com/user/repo

# shell integration
try init
```

For shell integration:

```bash
eval "$(try init)"
```
