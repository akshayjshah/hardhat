package main

import (
	"fmt"

	"github.com/akshayjshah/hardhat/internal/git"
	"github.com/akshayjshah/hardhat/internal/log"
	"github.com/akshayjshah/hardhat/internal/project"
)

func main() {
	logger := log.New()
	repo, err := git.New(logger)
	if err != nil {
		panic(err)
	}
	proj, err := project.New(logger, repo)
	if err != nil {
		panic(err)
	}
	diff, err := proj.RecursiveDiff("HEAD")
	if err != nil {
		panic(err)
	}
	fmt.Println(diff)
}
