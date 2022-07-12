// Copyright (C) 2022 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

// Package pathtools provides filepath-related functions used by multiple other
// packages.
package pathtools

import (
	"log"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ReplaceTilde takes a filepath and replaces a leading tilde with the current
// user's home directory.
func ReplaceTilde(filePath string) (string, error) {
	if strings.HasPrefix(filePath, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", errors.Wrap(err, "get current user")
		}
		filePath = filepath.Join(usr.HomeDir, filePath[1:])
	}
	return filePath, nil
}

// MustReplaceTilde does the same thing as ReplaceTilde, except that it panics
// if there is an error. Useful for testing.
func MustReplaceTilde(filePath string) string {
	filePath, err := ReplaceTilde(filePath)
	if err != nil {
		log.Fatal(err)
	}
	return filePath
}
