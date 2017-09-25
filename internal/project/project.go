package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/akshayjshah/hardhat/internal/git"
	"github.com/akshayjshah/hardhat/internal/log"
)

// deps is a portion of the information returned by "go list -json".
type deps struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
	Deps         []string
}

// Diff FIXME
type Diff struct {
	Files    []string
	Packages []string

	recursive bool
}

func (d Diff) String() string {
	if len(d.Files) == 0 && len(d.Packages) == 0 {
		return "No changes."
	}

	buf := bytes.NewBuffer(nil)
	if len(d.Files) == 0 {
		buf.WriteString("No changed files.\n")
	} else {
		fmt.Fprintf(buf, "%d changed files:\n", len(d.Files))
		for _, f := range d.Files {
			fmt.Fprintf(buf, "\t%s\n", f)
		}
	}
	if len(d.Packages) == 0 {
		buf.WriteString("No affected packages.\n")
	} else {
		if d.recursive {
			fmt.Fprintf(buf, "%d affected packages:\n", len(d.Packages))
		} else {
			fmt.Fprintf(buf, "%d directly affected packages:\n", len(d.Packages))
		}
		for _, p := range d.Packages {
			fmt.Fprintf(buf, "\t%s\n", p)
		}
	}
	return strings.TrimSpace(buf.String())
}

// Project FIXME
type Project struct {
	logger *log.Logger
	repo   *git.Repository
	root   string
}

// New FIXME
func New(logger *log.Logger, r *git.Repository) (*Project, error) {
	p := &Project{
		logger: logger,
		repo:   r,
	}
	if err := p.setRoot(); err != nil {
		return nil, fmt.Errorf("can't determine root package: %v", err)
	}
	return p, nil
}

// Root FIXME
func (p *Project) Root() string { return p.root }

// Diff FIXME
func (p *Project) Diff(since string) (Diff, error) {
	raw, err := p.repo.Diff(since)
	if err != nil {
		return Diff{}, err
	}
	return p.processDiff(raw), nil
}

// RecursiveDiff FIXME
func (p *Project) RecursiveDiff(since string) (Diff, error) {
	base, err := p.Diff(since)
	if err != nil {
		return Diff{}, err
	}

	g, err := p.graph()
	if err != nil {
		return Diff{}, fmt.Errorf("can't build project's import graph: %v", err)
	}

	affected := make(map[string]struct{})
	for _, p := range base.Packages {
		affected[p] = struct{}{}
		for _, dep := range g[p] {
			affected[dep] = struct{}{}
		}
	}

	base.Packages = make([]string, 0, len(affected))
	for pkg := range affected {
		base.Packages = append(base.Packages, pkg)
	}
	sort.Strings(base.Packages)
	base.recursive = true
	return base, nil
}

// All FIXME
func (p *Project) All() (Diff, error) {
	raw, err := p.repo.All()
	if err != nil {
		return Diff{}, err
	}
	return p.processDiff(raw), nil
}

func (p *Project) processDiff(raw git.Diff) Diff {
	var d Diff
	d.Files = raw.Modified

	dirs := make(map[string]struct{})
	for _, f := range raw.Deleted {
		dirs[filepath.Dir(f)] = struct{}{}
	}
	for _, f := range raw.Modified {
		dirs[filepath.Dir(f)] = struct{}{}
	}

	for dir := range dirs {
		pkg, err := build.Import(dir, p.repo.Root(), build.ImportComment)
		if err != nil {
			p.logger.Debugf("can't import directory %q: %v", dir, err)
		}
		d.Packages = append(d.Packages, filepath.Join(p.Root(), pkg.ImportPath))
	}
	sort.Strings(d.Packages)
	return d
}

func (p *Project) setRoot() error {
	pkg, err := build.ImportDir(p.repo.Root(), build.ImportComment)
	if err == nil && pkg.ImportPath != "" {
		p.root = pkg.ImportPath
		p.logger.Debugf("found root package %q", p.root)
		return nil
	}

	p.logger.Debugf("no Go source files in repository root, guessing root package from path")
	importPath, err := filepath.Rel(filepath.Join(build.Default.GOPATH, "src"), p.repo.Root())
	if err != nil {
		return fmt.Errorf("can't find repository path relative to $GOPATH: %v", err)
	}
	p.logger.Debugf("assuming root package is %q", importPath)
	p.root = importPath
	return nil
}

func (p *Project) graph() (map[string][]string, error) {
	out := bytes.NewBuffer(nil)
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = p.repo.Root()
	cmd.Stderr, cmd.Stdout = out, out
	if err := cmd.Run(); err != nil {
		p.logger.Debugf(out.String())
		return nil, err
	}

	dependencies := make(map[string][]string)
	dec := json.NewDecoder(out)
	for dec.More() {
		var d deps
		if err := dec.Decode(&d); err != nil {
			return nil, err
		}
		d.Deps = append(d.Deps, d.Imports...)
		d.Deps = append(d.Deps, d.TestImports...)
		d.Deps = append(d.Deps, d.XTestImports...)
		for _, pkg := range d.Deps {
			dependencies[pkg] = append(dependencies[pkg], d.ImportPath)
		}
	}
	return dependencies, nil
}
