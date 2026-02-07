package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
)

const version = "0.1.0"
const scriptWarning = "# if you can read this, run via eval: eval \"$(try exec)\""

var (
	errCancelled  = errors.New("cancelled")
	errRenderOnly = errors.New("render-only")
)

type cli struct {
	Path           string           `name:"path" env:"TRY_PATH" help:"Base path for tries." default:"~/src/tries"`
	NoColors       bool             `name:"no-colors" help:"Disable colors."`
	NoExpandTokens bool             `name:"no-expand-tokens" help:"Compatibility flag (treated as no-colors)."`
	Version        kong.VersionFlag `name:"version" help:"Show version."`

	Init     initCmd     `cmd:"" help:"Print shell function wrapper."`
	Clone    cloneCmd    `cmd:"" help:"Clone repository into a dated try dir."`
	Worktree worktreeCmd `cmd:"" help:"Create git worktree in a dated try dir."`
	Exec     execCmd     `cmd:"" help:"Run selector and print shell script."`
}

type initCmd struct {
	Path string `arg:"" optional:"" name:"path" help:"Optional tries path."`
}

type cloneCmd struct {
	URL  string `arg:"" name:"url" help:"Git URL." required:""`
	Name string `arg:"" optional:"" name:"name" help:"Custom directory name."`
}

type worktreeCmd struct {
	Name string `arg:"" optional:"" name:"name" help:"Worktree name (default: current dir name)."`
}

type execCmd struct {
	AndType    string `name:"and-type" hidden:""`
	AndExit    bool   `name:"and-exit" hidden:""`
	AndKeys    string `name:"and-keys" hidden:""`
	AndConfirm string `name:"and-confirm" hidden:""`

	Query []string `arg:"" optional:"" name:"query" help:"Search query, or git URL shorthand."`
}

type execOptions struct {
	NoColors   bool
	AndType    string
	AndExit    bool
	AndKeys    []string
	AndConfirm string
}

type compatOverrides struct {
	andType    *string
	andExit    bool
	andKeys    *string
	andConfirm *string
}

func main() {
	c := cli{}
	args := normalizeArgs(os.Args[1:])
	args, compat := extractCompatFlags(args)
	parser, err := kong.New(&c,
		kong.Name("try"),
		kong.Description("try - ephemeral workspace manager"),
		kong.Vars{"version": version},
		kong.UsageOnError(),
	)
	if err != nil {
		fail(err)
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		fail(err)
	}

	basePath, err := expandPath(c.Path)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		fail(err)
	}

	noColors := c.NoColors || c.NoExpandTokens || os.Getenv("NO_COLOR") != ""
	setNoColors(noColors)

	opts := execOptions{
		NoColors:   noColors,
		AndType:    c.Exec.AndType,
		AndExit:    c.Exec.AndExit,
		AndKeys:    parseTestKeys(c.Exec.AndKeys),
		AndConfirm: c.Exec.AndConfirm,
	}
	if compat.andType != nil {
		opts.AndType = *compat.andType
	}
	if compat.andKeys != nil {
		opts.AndKeys = parseTestKeys(*compat.andKeys)
	}
	if compat.andConfirm != nil {
		opts.AndConfirm = *compat.andConfirm
	}
	if compat.andExit {
		opts.AndExit = true
	}

	var script []string
	cmd := ctx.Command()
	switch {
	case cmd == "init":
		initPath := basePath
		if strings.TrimSpace(c.Init.Path) != "" {
			initPath, err = expandPath(c.Init.Path)
			if err != nil {
				fail(err)
			}
		}
		fmt.Println(initScript(initPath))
		return
	case strings.HasPrefix(cmd, "clone "):
		script, err = runClone(basePath, c.Clone.URL, c.Clone.Name)
	case strings.HasPrefix(cmd, "worktree "):
		script, err = runWorktree(basePath, c.Worktree.Name)
	case strings.HasPrefix(cmd, "exec "):
		script, err = runExec(basePath, strings.Join(c.Exec.Query, " "), opts)
	default:
		script, err = runExec(basePath, "", opts)
	}
	if err != nil {
		switch {
		case errors.Is(err, errRenderOnly):
			return
		case errors.Is(err, errCancelled):
			fmt.Fprintln(os.Stdout, "Cancelled.")
			os.Exit(1)
		default:
			fail(err)
		}
	}
	emitScript(script)
}

func runExec(basePath, query string, opts execOptions) ([]string, error) {
	query = strings.TrimSpace(query)
	if query != "" {
		parts := strings.Fields(query)
		if len(parts) > 0 {
			switch parts[0] {
			case "cd":
				query = strings.TrimSpace(strings.Join(parts[1:], " "))
				parts = strings.Fields(query)
			case "clone":
				if len(parts) < 2 {
					return nil, fmt.Errorf("git URI required for clone")
				}
				custom := ""
				if len(parts) > 2 {
					custom = strings.Join(parts[2:], "-")
				}
				return runClone(basePath, parts[1], custom)
			case "worktree":
				name := ""
				if len(parts) > 1 {
					name = strings.Join(parts[1:], "-")
				}
				return runWorktree(basePath, name)
			}
		}
	}

	if query != "" {
		parts := strings.Fields(query)
		if len(parts) > 0 {
			if parts[0] == "." || strings.HasPrefix(parts[0], "./") {
				return runDotWorktree(basePath, parts)
			}
			if isGitURI(parts[0]) {
				name := ""
				if len(parts) > 1 {
					name = strings.Join(parts[1:], "-")
				}
				return runClone(basePath, parts[0], name)
			}
		}
	}

	res, err := runSelector(basePath, query, selectorOptions{
		AndType:    opts.AndType,
		AndExit:    opts.AndExit,
		AndKeys:    opts.AndKeys,
		AndConfirm: opts.AndConfirm,
	})
	if err != nil {
		return nil, err
	}
	switch res.Kind {
	case "cd":
		return scriptCD(res.Path), nil
	case "mkdir":
		return scriptMkdirCD(res.Path), nil
	case "rename":
		return scriptRename(res.BasePath, res.OldName, res.NewName), nil
	case "delete":
		return scriptDelete(res.DeleteNames, res.BasePath), nil
	default:
		return nil, errCancelled
	}
}

func runDotWorktree(basePath string, parts []string) ([]string, error) {
	repoArg := parts[0]
	custom := ""
	if len(parts) > 1 {
		custom = strings.Join(parts[1:], "-")
	}
	if repoArg == "." && strings.TrimSpace(custom) == "" {
		return nil, fmt.Errorf("'try .' requires a name argument")
	}
	repoDir, err := filepath.Abs(repoArg)
	if err != nil {
		return nil, err
	}
	base := custom
	if base == "" {
		base = filepath.Base(repoDir)
	}
	base = sanitizeName(base)
	if base == "" {
		base = "worktree"
	}
	today := time.Now().Format("2006-01-02")
	dirName := uniqueDirName(basePath, fmt.Sprintf("%s-%s", today, base))
	target := filepath.Join(basePath, dirName)
	if fileOrDirExists(filepath.Join(repoDir, ".git")) {
		return scriptWorktree(target, repoDir), nil
	}
	return scriptMkdirCD(target), nil
}

func runClone(basePath, rawURL, name string) ([]string, error) {
	dirName, err := cloneDirName(basePath, rawURL, name)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(basePath, dirName)
	return scriptClone(target, rawURL), nil
}

func runWorktree(basePath, name string) ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	base := strings.TrimSpace(name)
	if base == "" {
		base = filepath.Base(wd)
	}
	base = sanitizeName(base)
	if base == "" {
		base = "worktree"
	}
	today := time.Now().Format("2006-01-02")
	dirName := uniqueDirName(basePath, fmt.Sprintf("%s-%s", today, base))
	target := filepath.Join(basePath, dirName)
	return scriptWorktree(target, wd), nil
}

func emitScript(cmds []string) {
	fmt.Println(scriptWarning)
	for i, cmd := range cmds {
		if i == len(cmds)-1 {
			fmt.Println(cmd)
			continue
		}
		fmt.Printf("%s && \\\n", cmd)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"exec"}
	}
	for _, a := range args {
		if a == "--help" || a == "-h" || a == "--version" || a == "-v" {
			return args
		}
	}
	known := map[string]struct{}{
		"init": {}, "clone": {}, "worktree": {}, "exec": {}, "help": {},
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if a == "--path" || a == "--and-type" || a == "--and-keys" || a == "--and-confirm" {
			i++
			continue
		}
		if strings.HasPrefix(a, "--path=") || strings.HasPrefix(a, "--and-type=") || strings.HasPrefix(a, "--and-keys=") || strings.HasPrefix(a, "--and-confirm=") {
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		if _, ok := known[a]; ok {
			return args
		}
		out := make([]string, 0, len(args)+1)
		out = append(out, args[:i]...)
		out = append(out, "exec")
		out = append(out, args[i:]...)
		return out
	}
	return append(args, "exec")
}

func parseTestKeys(spec string) []string {
	if spec == "" {
		return nil
	}
	useTokenMode := strings.Contains(spec, ",") || isAllUpperToken(spec)
	if !useTokenMode {
		keys := make([]string, 0, len(spec))
		for i := 0; i < len(spec); {
			if i+2 < len(spec) && spec[i] == 0x1b && spec[i+1] == '[' {
				keys = append(keys, spec[i:i+3])
				i += 3
				continue
			}
			keys = append(keys, string(spec[i]))
			i++
		}
		return keys
	}

	tokens := strings.Split(spec, ",")
	keys := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		u := strings.ToUpper(t)
		switch u {
		case "UP":
			keys = append(keys, "\x1b[A")
		case "DOWN":
			keys = append(keys, "\x1b[B")
		case "LEFT":
			keys = append(keys, "\x1b[D")
		case "RIGHT":
			keys = append(keys, "\x1b[C")
		case "ENTER":
			keys = append(keys, "\r")
		case "ESC":
			keys = append(keys, "\x1b")
		case "BACKSPACE":
			keys = append(keys, "\x7f")
		case "CTRL-A", "CTRLA":
			keys = append(keys, "\x01")
		case "CTRL-B", "CTRLB":
			keys = append(keys, "\x02")
		case "CTRL-C", "CTRLC":
			keys = append(keys, "\x03")
		case "CTRL-D", "CTRLD":
			keys = append(keys, "\x04")
		case "CTRL-E", "CTRLE":
			keys = append(keys, "\x05")
		case "CTRL-F", "CTRLF":
			keys = append(keys, "\x06")
		case "CTRL-H", "CTRLH":
			keys = append(keys, "\x08")
		case "CTRL-K", "CTRLK":
			keys = append(keys, "\x0b")
		case "CTRL-N", "CTRLN":
			keys = append(keys, "\x0e")
		case "CTRL-P", "CTRLP":
			keys = append(keys, "\x10")
		case "CTRL-R", "CTRLR":
			keys = append(keys, "\x12")
		case "CTRL-T", "CTRLT":
			keys = append(keys, "\x14")
		case "CTRL-W", "CTRLW":
			keys = append(keys, "\x17")
		default:
			if strings.HasPrefix(strings.ToUpper(t), "TYPE=") {
				v := t[len("TYPE="):]
				for _, ch := range v {
					keys = append(keys, string(ch))
				}
			} else if len(t) == 1 {
				keys = append(keys, t)
			}
		}
	}
	return keys
}

func extractCompatFlags(args []string) ([]string, compatOverrides) {
	out := make([]string, 0, len(args))
	ov := compatOverrides{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--and-exit":
			ov.andExit = true
		case a == "--and-type":
			if i+1 < len(args) {
				v := args[i+1]
				ov.andType = &v
				i++
			}
		case strings.HasPrefix(a, "--and-type="):
			v := strings.TrimPrefix(a, "--and-type=")
			ov.andType = &v
		case a == "--and-keys":
			if i+1 < len(args) {
				v := args[i+1]
				ov.andKeys = &v
				i++
			}
		case strings.HasPrefix(a, "--and-keys="):
			v := strings.TrimPrefix(a, "--and-keys=")
			ov.andKeys = &v
		case a == "--and-confirm":
			if i+1 < len(args) {
				v := args[i+1]
				ov.andConfirm = &v
				i++
			}
		case strings.HasPrefix(a, "--and-confirm="):
			v := strings.TrimPrefix(a, "--and-confirm=")
			ov.andConfirm = &v
		default:
			out = append(out, a)
		}
	}
	return out, ov
}

func isAllUpperToken(s string) bool {
	for _, r := range s {
		if r == '-' {
			continue
		}
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return s != ""
}
