// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

// Package opsys provides an interface OS for interacting with the running
// operating system, both with the filesystem and with Mac OS commands.
package opsys

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

//go:generate mockgen -destination=mock_opsys/mock_opsys.go github.com/tagatac/bagoup/opsys OS

type (
	// OS interacts with the local filesystem and operating system.
	OS interface {
		afero.Fs
		// FileAccess checks if the binary has access to the given file path.
		FileAccess(fp string) error
		// FileExist checks if the given path already exists.
		FileExist(fp string) (bool, error)
		// GetMacOSVersion checks the version of the current operating system,
		// assuming it is Mac OS.
		GetMacOSVersion() (*semver.Version, error)
		// GetContactMap gets a map of vcards indexed by phone numbers and email
		// addresses specified in those cards, from the vcard file at the given
		// path.
		GetContactMap(path string) (map[string]*vcard.Card, error)
		CopyFile(src, dstDir string) error
		RmTempDir() error
		HEIC2JPG(src string) (string, error)
		NewOutFile(filePath string, isPDF, includePPA bool) (OutFile, error)
	}

	opSys struct {
		afero.Fs
		osStat      func(string) (os.FileInfo, error)
		execCommand func(string, ...string) *exec.Cmd
		tempDir     string
	}
)

// NewOS returns an OS from a given filesystem, os Stat, and exec Command.
func NewOS(fs afero.Fs, osStat func(string) (os.FileInfo, error), execCommand func(string, ...string) *exec.Cmd) OS {
	return &opSys{Fs: fs, osStat: osStat, execCommand: execCommand}
}

func (s opSys) FileAccess(fp string) error {
	f, err := s.Open(fp)
	if err != nil {
		return errors.Wrapf(err, "open file %q", fp)
	}
	f.Close()
	return nil
}

func (s opSys) FileExist(fp string) (bool, error) {
	_, err := s.osStat(fp)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, errors.Wrapf(err, "check existence of file %q", fp)
}

func (s opSys) GetMacOSVersion() (*semver.Version, error) {
	cmd := s.execCommand("sw_vers", "-productVersion")
	o, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "call sw_vers")
	}
	vstr := strings.TrimSuffix(string(o), "\n")
	v, err := semver.NewVersion(vstr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse semantic version %q", vstr)
	}
	return v, nil
}

func (s opSys) GetContactMap(contactsFilePath string) (map[string]*vcard.Card, error) {
	f, err := s.Open(contactsFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := vcard.NewDecoder(f)
	contactMap := map[string]*vcard.Card{}
	for {
		card, err := dec.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrapf(err, "decode vcard")
		}
		phones := card.Values(vcard.FieldTelephone)
		for i, phone := range phones {
			phones[i] = sanitizePhone(phone)
		}
		phonesAndEmails := append(phones, card.Values(vcard.FieldEmail)...)
		for _, phoneOrEmail := range phonesAndEmails {
			if c, ok := contactMap[phoneOrEmail]; ok {
				log.Printf("WARN: multiple contacts %q and %q share the same phone or email %q", c.PreferredValue(vcard.FieldFormattedName), card.PreferredValue(vcard.FieldFormattedName), phoneOrEmail)
			}
			contactMap[phoneOrEmail] = &card
		}
	}
	return contactMap, nil
}

// Adapted from https://stackoverflow.com/a/44009184/5403337
func sanitizePhone(dirty string) string {
	return strings.Map(
		func(r rune) rune {
			if strings.ContainsRune("()-", r) || unicode.IsSpace(r) {
				return -1
			}
			return r
		},
		dirty,
	)
}

func (s opSys) CopyFile(src, dstDir string) error {
	fin, err := s.Open(src)
	if err != nil {
		return errors.Wrapf(err, "open source file %q", src)
	}
	defer fin.Close()
	dst := filepath.Join(dstDir, filepath.Base(src))
	fout, err := s.Create(dst)
	if err != nil {
		return errors.Wrapf(err, "create destination file %q", dst)
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)
	return err
}

func (s *opSys) getTempDir() (string, error) {
	if s.tempDir != "" {
		return s.tempDir, nil
	}
	p, err := os.MkdirTemp("", "bagoup")
	if err != nil {
		return "", errors.Wrap(err, "create temporary directory")
	}
	s.tempDir = p
	return p, nil
}

func (s *opSys) RmTempDir() error {
	if s.tempDir == "" {
		return nil
	}
	if err := s.RemoveAll(s.tempDir); err != nil {
		log.Printf("ERROR: failed to remove temporary directory %q: %s\n", s.tempDir, err)
		return errors.Wrapf(err, "remove temporary directory %q", s.tempDir)
	}
	s.tempDir = ""
	return nil
}
