package cmd

import (
	"fmt"

	"github.com/akshayjshah/hardhat/internal/hhlog"
	"github.com/akshayjshah/hardhat/internal/project"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type test struct {
	p      *project.Project
	logger *hhlog.Logger

	verbose   bool
	all       bool
	direct    bool
	base      string
	race      bool
	cover     bool
	covermode string
	list      string
	only      string // go test -run
	bench     string
}

func addTest(app *kingpin.Application, p *project.Project, l *hhlog.Logger) {
	t := &test{p: p, logger: l}
	cmd := app.Command("test", "Run unit tests.").Action(t.run)
	cmd.Flag("verbose", "Increase output verbosity.").
		Short('v').
		BoolVar(&t.verbose)
	cmd.Flag("direct", "Include only directly modified packages.").
		Short('d').
		BoolVar(&t.direct)
	cmd.Flag("base", "Commitish to compare against.").
		Default("origin/master").
		Short('b').
		StringVar(&t.base)
	cmd.Flag("all", "Run tests for all packages.").
		Short('a').
		BoolVar(&t.all)
	cmd.Flag("race", "Enable the race detector.").
		Short('r').
		BoolVar(&t.race)
	cmd.Flag("cover", "Enable coverage reporting.").
		Short('c').
		BoolVar(&t.cover)
	cmd.Flag("covermode", "Coverage calculation mode.").
		EnumVar(&t.covermode, "set", "count", "atomic")
	cmd.Flag("list", "List tests matching a regexp without running them.").
		StringVar(&t.list)
	cmd.Flag("run", "Run only tests matching a regexp.").
		StringVar(&t.only)
	cmd.Flag("bench", "Also run benchmarks matching a regexp, including memory profiling.").
		StringVar(&t.bench)

}

func (t *test) run(_ *kingpin.ParseContext) error {
	var d project.Diff
	var err error
	if t.all {
		d, err = t.p.All()
	} else if t.direct {
		d, err = t.p.Diff(t.base)
	} else {
		d, err = t.p.RecursiveDiff(t.base)
	}
	if err != nil {
		return t.logger.Annotate(err)
	}
	return t.test(d)
}

func (t *test) test(d project.Diff) error {
	args := []string{"test"}
	if t.verbose {
		args = append(args, "-v")
	}
	if t.race {
		args = append(args, "-race")
	}
	if t.cover {
		args = append(args, "-cover")
	}
	if t.covermode != "" {
		args = append(args, fmt.Sprintf("-%s", t.covermode))
	}
	if t.list != "" {
		args = append(args, "-list", t.list)
	}
	if t.only != "" {
		args = append(args, "-run", t.only)
	}
	if t.bench != "" {
		args = append(args, "-bench", t.bench)
	}

	pkgs := make([]string, 0, len(d.Packages))
	for _, pd := range d.Packages {
		if pd.Status == project.StatusModified {
			pkgs = append(pkgs, pd.Path)
		}
	}
	if len(pkgs) == 0 {
		t.logger.Printf("No packages need to be tested.")
		return nil
	}
	args = append(args, pkgs...)
	if err := t.p.Exec("go", args...); err != nil {
		return t.logger.Annotate(err)
	}
	return nil
}
