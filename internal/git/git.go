// Package git shells out to gather information about the current repository.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/akshayjshah/hardhat/internal/hhlog"
)

// A Diff enumerates the files changed between two points in history.
type Diff struct {
	Deleted  []string
	Modified []string
}

// A Repository offers access to a handful of useful git commands.
type Repository struct {
	logger *hhlog.Logger
	root   string
}

// New initializes and returns a Repository.
func New(logger *hhlog.Logger) (*Repository, error) {
	repo := &Repository{logger: logger}
	if err := repo.setRoot(); err != nil {
		return nil, err
	}
	return repo, nil
}

// Root returns the absolute path to the repository root.
func (r *Repository) Root() string { return r.root }

// Canonicalize converts the supplied commitish to a SHA1.
func (r *Repository) Canonicalize(commitish string) (string, error) {
	sha, err := r.run("rev-parse", commitish)
	if err != nil {
		return "", fmt.Errorf("can't resolve %q to SHA1: %v", commitish, err)
	}
	r.logger.Debugf("resolved commitish %q to SHA1 %s", commitish, sha)
	return sha, nil
}

// Diff returns the paths of files changed since the supplied commitish,
// relative to the repository root.
func (r *Repository) Diff(since string) (Diff, error) {
	var diff Diff
	untracked, err := r.run(
		r.Root(),
		"ls-files",
		"--others",           // show untracked files
		"--exclude-standard", // honor standard .gitignores
	)
	if err != nil {
		return Diff{}, fmt.Errorf("can't find untracked files files: %v", err)
	}
	for _, u := range strings.Split(untracked, "\n") {
		if u != "" {
			diff.Modified = append(diff.Modified, u)
		}
	}

	modified, err := r.run(
		r.Root(),
		"diff",
		"--name-status", // print name and status
		"--no-renames",  // treat renames as a delete and an add
		"--ignore-submodules",
		since,
		"--", // compare against working tree
	)
	if err != nil {
		return Diff{}, fmt.Errorf("can't identify modified files: %v", err)
	}
	for _, u := range strings.Split(modified, "\n") {
		if u == "" {
			continue
		}
		fname := strings.Fields(u)[1]
		if strings.HasPrefix(u, "D") {
			diff.Deleted = append(diff.Deleted, fname)
			continue
		}
		diff.Modified = append(diff.Modified, fname)
	}

	sort.Strings(diff.Deleted)
	sort.Strings(diff.Modified)
	r.logger.Debugf("files deleted since %q: %v", since, diff.Deleted)
	r.logger.Debugf("files created or modified since %q: %v", since, diff.Modified)
	return diff, nil
}

// All returns all files in the repository, including untracked files,
// relative to the repository root.
func (r *Repository) All() (Diff, error) {
	var diff Diff
	all, err := r.run(
		r.Root(),
		"ls-files",
		"--cached",
		"--modified",
		"--others",
		"--exclude-standard", // honor standard .gitignores
	)
	if err != nil {
		return Diff{}, fmt.Errorf("can't list files: %v", err)
	}
	for _, f := range strings.Split(all, "\n") {
		if f != "" {
			diff.Modified = append(diff.Modified, f)
		}
	}
	r.logger.Debugf("found %d files in repository: %v", len(diff.Modified), diff.Modified)
	return diff, nil
}

func (r *Repository) setRoot() error {
	root, err := r.run("", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("can't determine repository root: %v", err)
	}
	r.logger.Debugf("repository root is %q", root)
	r.root = root
	return nil
}

func (r *Repository) run(cwd string, subcommand ...string) (string, error) {
	out := bytes.NewBuffer(nil)
	cmd := exec.Command("git", subcommand...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Stderr, cmd.Stdout = out, out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	result := string(bytes.TrimSpace(out.Bytes()))
	return result, nil
}
