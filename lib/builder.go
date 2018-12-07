package again

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"go/parser"
	"go/token"
)

type Builder interface {
	Build() error
	Binary() string
	Errors() string
	ShouldDepEnsure() bool
}

type builder struct {
	dir       string
	binary    string
	errors    string
	wd        string
	buildArgs []string
	importSet map[string]bool
}

func NewBuilder(dir string, bin string, wd string, buildArgs []string) Builder {
	if len(bin) == 0 {
		bin = "bin"
	}

	// does not work on Windows without the ".exe" extension
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(bin, ".exe") { // check if it already has the .exe extension
			bin += ".exe"
		}
	}

	return &builder{dir: dir, binary: bin, wd: wd, buildArgs: buildArgs, importSet: make(map[string]bool)}
}

func (b *builder) Binary() string {
	return b.binary
}

func (b *builder) Errors() string {
	return b.errors
}

func (b *builder) Build() error {
	args := append([]string{"go", "build", "-o", filepath.Join(b.wd, b.binary)}, b.buildArgs...)

	if b.ShouldDepEnsure() {
		depCommand := exec.Command("dep", "ensure")
		depCommand.Dir = b.dir
		out, _ := depCommand.CombinedOutput()
	
		if depCommand.ProcessState.Success() {
			b.errors = ""
		} else {
			b.errors = string(out)
		}

		if len(b.errors) > 0 {
			return fmt.Errorf(b.errors)
		}
	}

	command := exec.Command(args[0], args[1:]...)

	command.Dir = b.dir

	output, err := command.CombinedOutput()

	if command.ProcessState.Success() {
		b.errors = ""
	} else {
		b.errors = string(output)
	}

	if len(b.errors) > 0 {
		return fmt.Errorf(b.errors)
	}

	return err
}

func (b *builder) ShouldDepEnsure () bool {
	fileset := token.NewFileSet()

	pkgs, _ := parser.ParseDir(fileset, b.wd, nil, parser.ImportsOnly)

	var hasNewDependency bool = false
	for _, pkg := range pkgs {
		for _, astFile := range pkg.Files {
			for _, importSpec := range astFile.Imports {
				_, found := b.importSet[importSpec.Path.Value]
				if !found {
					hasNewDependency = true
				}
				b.importSet[importSpec.Path.Value] = true
			}
		}
	}
	return hasNewDependency
}