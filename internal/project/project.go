// Package project models a git repository of Go source code.
package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/akshayjshah/hardhat/internal/git"
	"github.com/akshayjshah/hardhat/internal/hhlog"
)

// deps is a portion of the information returned by "go list -json".
type deps struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
	Deps         []string
}

// Status describes the state of a file or package relative to a previous
// commit.
type Status uint8

// Relative to a previous commit, each file or package is either unchanged,
// modified, or deleted.
const (
	StatusUnknown Status = iota
	StatusUnchanged
	StatusModified
	StatusDeleted
)

func (s Status) String() string {
	switch s {
	case StatusUnchanged:
		return "-"
	case StatusModified:
		return "M"
	case StatusDeleted:
		return "D"
	default:
		return "?"
	}
}

// MarshalText implements encoding.TextMarshaler.
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// A PathDiff describes the state of a single file relative to a previous
// commit.
type PathDiff struct {
	Status Status `json:"status"`
	Path   string `json:"path"`
}

func (pd PathDiff) less(other PathDiff) bool {
	if pd.Status < other.Status {
		return true
	}
	return pd.Path < other.Path
}

// A Diff identifies the files and packages modified since the base commit.
// Recursive diffs also include packages that depend on modified code.
type Diff struct {
	Files    []PathDiff `json:"files"`
	Packages []PathDiff `json:"packages"`

	recursive bool
}

func (d Diff) String() string {
	if len(d.Files)+len(d.Packages) == 0 {
		return "No changes."
	}

	buf := bytes.NewBuffer(nil)
	if len(d.Files) == 0 {
		buf.WriteString("No modified or deleted files.\n")
	} else {
		fmt.Fprintf(buf, "%d modified or deleted files:\n", len(d.Files))
		for _, pd := range d.Files {
			fmt.Fprintf(buf, "\t%s\t%s\n", pd.Status, pd.Path)
		}
	}

	if len(d.Packages) == 0 {
		buf.WriteString("No affected packages.\n")
	} else {
		fmt.Fprintf(buf, "%d modified or deleted packages:\n", len(d.Packages))
		for _, pd := range d.Packages {
			fmt.Fprintf(buf, "\t%s\t%s\n", pd.Status, pd.Path)
		}
	}
	return strings.TrimSpace(buf.String())
}

// A Project represents a git repository of Go source code.
type Project struct {
	logger *hhlog.Logger
	repo   *git.Repository
	root   string
}

// New constructs a project.
func New(logger *hhlog.Logger, r *git.Repository) (*Project, error) {
	p := &Project{
		logger: logger,
		repo:   r,
	}
	if err := p.setRoot(); err != nil {
		return nil, fmt.Errorf("can't determine root package: %v", err)
	}
	return p, nil
}

// Root returns the Go import path of the project's root package.
func (p *Project) Root() string { return p.root }

// Diff identifies the files and packages directly modified since the supplied
// commitish.
func (p *Project) Diff(since string) (Diff, error) {
	raw, err := p.repo.Diff(since)
	if err != nil {
		return Diff{}, err
	}
	return p.processDiff(raw), nil
}

// RecursiveDiff identifies the files and packages directly modified since the
// supplied commitish, along with any packages that depend on modified code.
func (p *Project) RecursiveDiff(since string) (Diff, error) {
	base, err := p.Diff(since)
	if err != nil {
		return Diff{}, err
	}

	g, err := p.graph()
	if err != nil {
		return Diff{}, fmt.Errorf("can't build project's import graph: %v", err)
	}

	affected := make(map[PathDiff]struct{})
	for _, pd := range base.Packages {
		affected[pd] = struct{}{}
		for _, dep := range g[pd.Path] {
			affected[PathDiff{StatusModified, dep}] = struct{}{}
		}
	}

	base.Packages = make([]PathDiff, 0, len(affected))
	for pkg := range affected {
		base.Packages = append(base.Packages, pkg)
	}
	sort.Slice(base.Packages, func(i, j int) bool {
		return base.Packages[i].less(base.Packages[j])
	})
	base.recursive = true
	return base, nil
}

// All identifies all the files and packages in the project.
func (p *Project) All() (Diff, error) {
	raw, err := p.repo.All()
	if err != nil {
		return Diff{}, err
	}
	return p.processDiff(raw), nil
}

// Exec executes a command, sending the output directly to standard out and
// standard error.
func (p *Project) Exec(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Dir = p.repo.Root()
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

func (p *Project) processDiff(raw git.Diff) Diff {
	d := Diff{
		Files:    make([]PathDiff, 0, len(raw.Modified)),
		Packages: make([]PathDiff, 0, len(raw.Deleted)),
	}
	for _, mod := range raw.Modified {
		d.Files = append(d.Files, PathDiff{StatusModified, mod})
	}
	for _, del := range raw.Deleted {
		d.Files = append(d.Files, PathDiff{StatusDeleted, del})
	}

	dirs := make(map[string]struct{})
	for _, f := range raw.Deleted {
		dirs[filepath.Dir(f)] = struct{}{}
	}
	for _, f := range raw.Modified {
		dirs[filepath.Dir(f)] = struct{}{}
	}

	for dir := range dirs {
		if !exists(dir) {
			d.Packages = append(d.Packages, PathDiff{StatusDeleted, filepath.Join(p.Root(), dir)})
			continue
		}

		if filepath.Base(dir) == "testdata" {
			// If a testdata directory is changed, assume that it affects only the
			// containing package.
			dir = filepath.Dir(dir)
		}

		pkg, err := build.Import(fmt.Sprintf("./%s", dir), p.repo.Root(), build.ImportComment)
		if err != nil {
			// Not all directories are Go packages.
			p.logger.Debugf("Go tool can't import directory %q: %v", dir, err)
			continue
		}

		if dir == "." {
			d.Packages = append(d.Packages, PathDiff{StatusModified, p.Root()})
		} else {
			d.Packages = append(d.Packages, PathDiff{StatusModified, pkg.ImportPath})
		}
	}
	sort.Slice(d.Packages, func(i, j int) bool {
		return d.Packages[i].less(d.Packages[j])
	})
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

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
