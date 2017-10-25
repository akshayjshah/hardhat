package cmd

import (
	"encoding/json"

	"github.com/akshayjshah/hardhat/internal/hhlog"
	"github.com/akshayjshah/hardhat/internal/project"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type status struct {
	p      *project.Project
	logger *hhlog.Logger

	direct bool
	base   string
	json   bool
}

func addStatus(app *kingpin.Application, p *project.Project, l *hhlog.Logger) {
	s := &status{p: p, logger: l}
	cmd := app.Command("status", "Show project status.").Action(s.run)
	cmd.Flag("direct", "Include only directly modified packages.").
		Short('d').
		BoolVar(&s.direct)
	cmd.Flag("base", "Commitish to compare against.").
		Default("origin/master").
		Short('b').
		StringVar(&s.base)
	cmd.Flag("json", "Format output as JSON.").
		BoolVar(&s.json)
}

func (s *status) run(_ *kingpin.ParseContext) error {
	var d project.Diff
	var err error
	if s.direct {
		d, err = s.p.Diff(s.base)
	} else {
		d, err = s.p.RecursiveDiff(s.base)
	}
	if err != nil {
		return s.logger.Annotate(err)
	}
	if s.json {
		bs, err := json.Marshal(d)
		if err != nil {
			return s.logger.Annotate(err)
		}
		s.logger.Printf(string(bs))
	} else {
		s.logger.Printf(d.String())
	}
	return nil
}
