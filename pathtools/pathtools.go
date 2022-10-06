// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

// Package pathtools provides filepath-related functions used by multiple other
// packages.
package pathtools

import (
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type (
	PathTools interface {
		// GetHomeDir returns the home directory of the running user (or the user
		// who originally ran bagop, in the case of a re-run).
		GetHomeDir() string
		// ReplaceTilde takes a filepath and replaces a leading tilde with the
		// current user's home directory.
		ReplaceTilde(filePath string) string
	}

	ptools struct {
		homeDir string
	}
)

func NewPathTools() (PathTools, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, errors.Wrap(err, "get current user")
	}
	return ptools{homeDir: usr.HomeDir}, nil
}

func NewPathToolsWithHomeDir(homeDir string) (PathTools, error) {
	if homeDir == "" {
		return nil, errors.New("re-run not possible: missing the home directory from the previous run - FIX: create a file .tildeexpansion with the expanded home directory from the previous run and place it at the root of the preserved-paths copied attachments directory (usually bagoup-attachments)")
	}
	return ptools{homeDir: homeDir}, nil
}

func (pt ptools) GetHomeDir() string { return pt.homeDir }

func (pt ptools) ReplaceTilde(filePath string) string {
	if strings.HasPrefix(filePath, "~") {
		filePath = filepath.Join(pt.GetHomeDir(), filePath[1:])
	}
	return filePath
}
