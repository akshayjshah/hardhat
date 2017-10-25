package cmd

import (
	"github.com/akshayjshah/hardhat/internal/git"
	"github.com/akshayjshah/hardhat/internal/hhlog"
	"github.com/akshayjshah/hardhat/internal/project"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// New builds the hardhat application.
func New(logger *hhlog.Logger) (*kingpin.Application, error) {
	repo, err := git.New(logger)
	if err != nil {
		return nil, err
	}
	proj, err := project.New(logger, repo)
	if err != nil {
		return nil, err
	}
	_ = proj
	app := kingpin.New("hardhat", "A git-centric Go build tool.")
	app.HelpFlag.Short('h')
	addStatus(app, proj, logger)
	addTest(app, proj, logger)
	return app, nil
}
