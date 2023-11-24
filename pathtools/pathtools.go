// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

// Package pathtools provides filepath-related functions used by multiple other
// packages.
package pathtools

import (
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

//go:generate mockgen -destination=mock_pathtools/mock_pathtools.go github.com/tagatac/bagoup/pathtools PathTools

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

func NewPathToolsWithHomeDir(homeDir string) PathTools {
	return ptools{homeDir: homeDir}
}

func (pt ptools) GetHomeDir() string { return pt.homeDir }

func (pt ptools) ReplaceTilde(filePath string) string {
	if strings.HasPrefix(filePath, "~") {
		filePath = filepath.Join(pt.GetHomeDir(), filePath[1:])
	}
	return filePath
}
