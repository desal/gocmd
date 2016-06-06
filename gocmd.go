package gocmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/desal/cmd"
	"github.com/desal/dsutil"
)

type empty struct{}
type set map[string]empty

var (
	stdLibs   set
	cacheLock sync.Mutex
)

type Context struct {
	output cmd.Output
	goPath []string
}

func New(output cmd.Output, goPath []string) *Context {
	result := &Context{output: output, goPath: goPath}
	result.checkCache()
	return result
}

func FromEnv(output cmd.Output) []string {
	envPath := os.Getenv("GOPATH")
	if len(envPath) == 0 {
		output.Error("GOPATH not set")
		os.Exit(1)
	}

	var goPath []string
	if runtime.GOOS == "windows" {
		goPath = strings.Split(envPath, ";")
	} else {
		goPath = strings.Split(envPath, ":")
	}

	if !dsutil.CheckPath(goPath[0]) {
		output.Error("First GOPATH element (%s) not found", goPath[0])
		os.Exit(1)
	}
	return goPath
}

func (c *Context) checkCache() {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	if stdLibs == nil {
		stdLibs = set{}
		ctx := cmd.NewContext("", c.output, cmd.Must)
		res, _ := ctx.Execf("go list std")

		for _, lib := range dsutil.SplitLines(res, true) {
			stdLibs[lib] = empty{}
		}
	}
}

func (c *Context) list(workingDir, pkgs string, cmdCtx *cmd.Context) (
	map[string]map[string]interface{}, error) {

	result := map[string]map[string]interface{}{}
	cmdRes, err := cmdCtx.Execf("go list -json %s", pkgs)
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
	cmdCtx := cmd.NewContext(workingDir, c.output, cmd.Warn)
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
	cmdCtx := cmd.NewContext(workingDir, c.output, cmd.Warn)
	_, err := cmdCtx.Execf("go install %s", pkgs)
	return err
}

func (c *Context) IsStdLib(importPath string) bool {
	_, ok := stdLibs[importPath]
	return ok
}

func (c *Context) Format(src string) (string, error) {
	cmdCtx := cmd.NewContext(".", c.output, cmd.Warn)
	r := bytes.NewReader([]byte(src))
	output, err := cmdCtx.PipeExecf(r, "gofmt")
	return output, err
}
