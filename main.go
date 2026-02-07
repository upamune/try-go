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
	errCancelled = errors.New("cancelled")
)

type cli struct {
	Path    string           `name:"path" env:"TRY_PATH" help:"Base path for tries." default:"~/src/tries"`
	Version kong.VersionFlag `name:"version" help:"Show version."`

	Init     initCmd     `cmd:"" help:"Print shell function wrapper."`
	Clone    cloneCmd    `cmd:"" help:"Clone repository into a dated try dir."`
	Worktree worktreeCmd `cmd:"" help:"Create git worktree in a dated try dir."`
	Exec     execCmd     `cmd:"" help:"Run selector and print shell script."`
}

type initCmd struct{}

type cloneCmd struct {
	URL  string `arg:"" name:"url" help:"Git URL." required:""`
	Name string `arg:"" optional:"" name:"name" help:"Custom directory name."`
}

type worktreeCmd struct {
	Name string `arg:"" optional:"" name:"name" help:"Worktree name (default: current dir name)."`
}

type execCmd struct {
	Query []string `arg:"" optional:"" name:"query" help:"Search query, or git URL shorthand."`
}

func main() {
	c := cli{}
	args := normalizeArgs(os.Args[1:])
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

	var script []string
	cmd := ctx.Command()
	switch {
	case cmd == "init":
		fmt.Println(initScript(basePath))
		return
	case strings.HasPrefix(cmd, "clone "):
		script, err = runClone(basePath, c.Clone.URL, c.Clone.Name)
	case strings.HasPrefix(cmd, "worktree "):
		script, err = runWorktree(basePath, c.Worktree.Name)
	case strings.HasPrefix(cmd, "exec "):
		script, err = runExec(basePath, strings.Join(c.Exec.Query, " "))
	default:
		script, err = runExec(basePath, "")
	}
	if err != nil {
		if errors.Is(err, errCancelled) {
			fmt.Fprintln(os.Stdout, "Cancelled.")
			os.Exit(1)
		}
		fail(err)
	}
	emitScript(script)
}

func runExec(basePath, query string) ([]string, error) {
	query = strings.TrimSpace(query)
	if query != "" {
		parts := strings.Fields(query)
		if len(parts) > 0 && isGitURI(parts[0]) {
			name := ""
			if len(parts) > 1 {
				name = strings.Join(parts[1:], "-")
			}
			return runClone(basePath, parts[0], name)
		}
	}

	res, err := runSelector(basePath, query)
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
		return scriptDelete(res.BasePath, res.DeleteNames), nil
	default:
		return nil, errCancelled
	}
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
		if a == "--help" || a == "-h" || a == "--version" {
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
		if a == "--path" {
			i++
			continue
		}
		if strings.HasPrefix(a, "--path=") {
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
