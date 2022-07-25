// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/opsys"
)

func (cfg configuration) bagoup() error {
	opts := cfg.Options
	if err := validatePaths(cfg.OS, opts); err != nil {
		return err
	}

	var err error
	if opts.MacOSVersion != nil {
		cfg.MacOSVersion, err = semver.NewVersion(*opts.MacOSVersion)
		if err != nil {
			return errors.Wrapf(err, "parse Mac OS version %q", *opts.MacOSVersion)
		}
	} else if cfg.MacOSVersion, err = cfg.OS.GetMacOSVersion(); err != nil {
		return errors.Wrap(err, "get Mac OS version - FIX: specify the Mac OS version from which chat.db was copied with the --mac-os-version option")
	}

	var contactMap map[string]*vcard.Card
	if opts.ContactsPath != nil {
		contactMap, err = cfg.OS.GetContactMap(*opts.ContactsPath)
		if err != nil {
			return errors.Wrapf(err, "get contacts from vcard file %q", *opts.ContactsPath)
		}
	}

	if err := cfg.ChatDB.Init(cfg.MacOSVersion); err != nil {
		return errors.Wrapf(err, "initialize the database for reading on Mac OS version %s", cfg.MacOSVersion.String())
	}

	cfg.HandleMap, err = cfg.ChatDB.GetHandleMap(contactMap)
	if err != nil {
		return errors.Wrap(err, "get handle map")
	}

	defer cfg.OS.RmTempDir()
	err = cfg.exportChats(contactMap)
	printResults(opts.ExportPath, cfg.Counts, time.Since(cfg.StartTime))
	if err != nil {
		return errors.Wrap(err, "export chats")
	}
	return cfg.OS.RmTempDir()
}

func validatePaths(s opsys.OS, opts options) error {
	if err := s.FileAccess(opts.DBPath); err != nil {
		return errors.Wrapf(err, "test DB file %q - FIX: %s", opts.DBPath, _readmeURL)
	}
	if exist, err := s.FileExist(opts.ExportPath); exist {
		return fmt.Errorf("export folder %q already exists - FIX: move it or specify a different export path with the --export-path option", opts.ExportPath)
	} else if err != nil {
		return errors.Wrapf(err, "check export path %q", opts.ExportPath)
	}
	return nil
}

func printResults(exportPath string, c counts, duration time.Duration) {
	var attachmentsString string
	for mimeType, count := range c.attachments {
		attachmentsString += fmt.Sprintf("\n\t%s: %d", mimeType, count)
	}
	if attachmentsString == "" {
		attachmentsString = "0"
	}
	var attachmentsEmbeddedString string
	for mimeType, count := range c.attachmentsEmbedded {
		attachmentsEmbeddedString += fmt.Sprintf("\n\t%s: %d", mimeType, count)
	}
	if attachmentsEmbeddedString == "" {
		attachmentsEmbeddedString = "0"
	}
	log.Printf(`%sBAGOUP RESULTS:
bagoup version: %s
Export folder: %q
Export files written: %d
Chats exported: %d
Messages exported: %d
Attachments referenced or embedded: %s
Attachments embedded: %s
Attachments missing (see warnings above): %d
HEIC conversions completed: %d
HEIC conversions failed (see warnings above): %d
Time elapsed: %s%s`,
		"\x1b[1m",
		_version,
		exportPath,
		c.files,
		c.chats,
		c.messages,
		attachmentsString,
		attachmentsEmbeddedString,
		c.attachmentsMissing,
		c.conversions,
		c.conversionsFailed,
		duration.String(),
		"\x1b[0m",
	)
}
