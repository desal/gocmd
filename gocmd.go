package gocmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/desal/cmd"
	"github.com/desal/dsutil"
	"github.com/desal/richtext"
)

//go:generate stringer -type Flag

type (
	empty     struct{}
	Flag      int
	stringSet map[string]empty

	Context struct {
		format   richtext.Format
		goPath   []string
		cmdFlags []cmd.Flag
	}
)

const (
	_ Flag = iota
	MustExit
	MustPanic
	Warn
	Verbose
	PassThrough
)

var (
	stdLibs   stringSet
	cacheLock sync.Mutex
	cmdFlags  = map[Flag]cmd.Flag{
		MustExit:  cmd.MustExit,
		MustPanic: cmd.MustPanic,
		Warn:      cmd.Warn,
		Verbose:   cmd.Verbose,
	}
)

func New(format richtext.Format, goPath []string, flags ...Flag) *Context {
	c := &Context{format: format, goPath: goPath}
	for _, flag := range flags {
		c.cmdFlags = append(c.cmdFlags, cmdFlags[flag])
	}

	if err := c.checkCache(); err != nil {
		panic(err)
	}
	return c
}

func EnvGoPath() ([]string, error) {
	envPath := os.Getenv("GOPATH")
	if len(envPath) == 0 {
		return nil, fmt.Errorf("GOPATH not set")
	}

	goPath := strings.Split(envPath, string(filepath.ListSeparator))

	if !dsutil.CheckPath(goPath[0]) {
		return nil, fmt.Errorf("First GOPATH element (%s) not found", goPath[0])
	}
	return goPath, nil
}

func (c *Context) checkCache() error {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	if stdLibs == nil {
		stdLibs = stringSet{}
		ctx := cmd.New("", c.format, c.cmdFlags...)
		res, _, err := ctx.Execf("go list std")
		if err != nil {
			return err
		}

		for _, lib := range dsutil.SplitLines(res, true) {
			stdLibs[lib] = empty{}
		}
	}
	return nil
}

func (c *Context) list(workingDir, pkgs string, cmdCtx *cmd.Context) (
	map[string]map[string]interface{}, error) {

	result := map[string]map[string]interface{}{}
	cmdRes, _, err := cmdCtx.Execf("go list -json %s", pkgs)
	if err != nil {
		return result, err
	}

	dec := json.NewDecoder(strings.NewReader(cmdRes))
	for {
		var e map[string]interface{}
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		importPathInt, ok := e["ImportPath"]
		if !ok {
			return nil, fmt.Errorf("Entry with missing ImportPath")
		}

		importPath, ok := importPathInt.(string)
		if !ok {
			return nil, fmt.Errorf("ImportPath not a string")
		}

		result[importPath] = e
	}

	return result, nil
}

func (c *Context) List(workingDir, pkgs string) (map[string]map[string]interface{}, error) {
	cmdCtx := cmd.New(workingDir, c.format, c.cmdFlags...)
	return c.list(workingDir, pkgs, cmdCtx)
}

func (c *Context) Dir(workingDir, pkg string) (string, bool) {
	for _, entry := range c.goPath {
		joined := filepath.Join(entry, "src", pkg)
		if dsutil.CheckPath(joined) {
			return joined, true
		}
	}
	return filepath.Join(c.goPath[0], "src", pkg), false
}

func (c *Context) Install(workingDir string, pkgs string) error {
	cmdCtx := cmd.New(workingDir, c.format, c.cmdFlags...)
	_, _, err := cmdCtx.Execf("go install %s", pkgs)
	return err
}

func (c *Context) IsStdLib(importPath string) bool {
	_, ok := stdLibs[importPath]
	return ok
}

func (c *Context) Format(src string) (string, error) {
	cmdCtx := cmd.New(".", c.format, c.cmdFlags...)
	r := bytes.NewReader([]byte(src))
	output, _, err := cmdCtx.PipeExecf(r, "gofmt")
	return output, err
}
