package main

import (
	"fmt"
	"os"

	"github.com/akshayjshah/hardhat/internal/cmd"
	"github.com/akshayjshah/hardhat/internal/hhlog"
)

func main() {
	logger := hhlog.New()
	c, err := cmd.New(logger)
	if err != nil {
		fmt.Println(logger.Annotate(err))
		os.Exit(1)
	}
	if _, err := c.Parse(os.Args[1:]); err != nil {
		logger.Printf(err.Error())
		os.Exit(1)
	}
}
