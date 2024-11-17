// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

// Package bagoup reads data from a Mac OS messsages chat database and exports
// it to text or PDF.
package bagoup

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/v2/chatdb"
	"github.com/tagatac/bagoup/v2/opsys"
	"github.com/tagatac/bagoup/v2/pathtools"
)

const (
	PreservedPathDir                = "bagoup-attachments"
	PreservedPathTildeExpansionFile = ".tildeexpansion"
)

const _readmeURL = "https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access"

type (
	configuration struct {
		Options
		opsys.OS
		chatdb.ChatDB
		pathtools.PathTools
		logDir          string
		macOSVersion    *semver.Version
		handleMap       map[int]string
		attachmentPaths map[int][]chatdb.Attachment
		counts
		startTime time.Time
		version   string
	}
	counts struct {
		files               int
		chats               int
		messages            int
		messagesInvalid     int
		attachments         map[string]int
		attachmentsCopied   map[string]int
		attachmentsEmbedded map[string]int
		attachmentsMissing  int
		conversions         int
		conversionsFailed   int
	}

	// Configuration is an interface for the bagoup command to run the meat of
	// bagoup.
	Configuration interface {
		// Run runs bagoup.
		Run() error
	}
)

// NewConfiguration returns an intitialized bagoup configuration.
func NewConfiguration(opts Options, s opsys.OS, cdb chatdb.ChatDB, ptools pathtools.PathTools, logDir string, startTime time.Time, version string) (Configuration, error) {
	if opts.AttachmentsPath != "/" {
		tef := filepath.Join(opts.AttachmentsPath, PreservedPathTildeExpansionFile)
		homeDir, err := s.ReadFile(tef)
		if err != nil {
			return nil, errors.Wrapf(err, "read tilde expansion file %q - POSSIBLE FIX: create a file .tildeexpansion with the expanded home directory from the previous run and place it at the root of the preserved-paths copied attachments directory (usually %q)", tef, PreservedPathDir)
		}
		ptools = pathtools.NewPathToolsWithHomeDir(string(homeDir))
	}
	return &configuration{
		Options:   opts,
		OS:        s,
		ChatDB:    cdb,
		PathTools: ptools,
		logDir:    logDir,
		counts: counts{
			attachments:         map[string]int{},
			attachmentsCopied:   map[string]int{},
			attachmentsEmbedded: map[string]int{},
		},
		startTime: startTime,
		version:   version,
	}, nil
}

func (cfg *configuration) Run() error {
	if err := cfg.validatePaths(); err != nil {
		return err
	}

	if err := cfg.OS.MkdirAll(cfg.logDir, os.ModePerm); err != nil {
		return errors.Wrap(err, "make log directory")
	}
	logFile, err := cfg.OS.Create(filepath.Join(cfg.logDir, "out.log"))
	if err != nil {
		return errors.Wrap(err, "create log file")
	}
	defer logFile.Close()
	w := log.Writer()
	log.SetOutput(io.MultiWriter(logFile, w))
	defer log.SetOutput(w)

	if cfg.Options.MacOSVersion != nil {
		cfg.macOSVersion, err = semver.NewVersion(*cfg.Options.MacOSVersion)
		if err != nil {
			return errors.Wrapf(err, "parse Mac OS version %q", *cfg.Options.MacOSVersion)
		}
	} else if cfg.macOSVersion, err = cfg.OS.GetMacOSVersion(); err != nil {
		return errors.Wrap(err, "get Mac OS version - FIX: specify the Mac OS version from which chat.db was copied with the --mac-os-version option")
	}

	var contactMap map[string]*vcard.Card
	if cfg.Options.ContactsPath != nil {
		contactMap, err = cfg.OS.GetContactMap(*cfg.Options.ContactsPath)
		if err != nil {
			return errors.Wrapf(err, "get contacts from vcard file %q", *cfg.Options.ContactsPath)
		}
	}

	if err := cfg.ChatDB.Init(cfg.macOSVersion); err != nil {
		return errors.Wrapf(err, "initialize the database for reading on Mac OS version %s", cfg.macOSVersion.String())
	}

	cfg.handleMap, err = cfg.ChatDB.GetHandleMap(contactMap)
	if err != nil {
		return errors.Wrap(err, "get handle map")
	}

	defer cfg.OS.RmTempDir()
	err = cfg.exportChats(contactMap)
	printResults(cfg.version, cfg.Options.ExportPath, cfg.counts, time.Since(cfg.startTime))
	if err != nil {
		return errors.Wrap(err, "export chats")
	}
	if err = cfg.writeTildeExpansionFile(); err != nil {
		return errors.Wrap(err, "write out tilde expansion file")
	}
	return cfg.OS.RmTempDir()
}

func (cfg *configuration) validatePaths() error {
	if err := cfg.OS.FileAccess(cfg.Options.DBPath); err != nil {
		return errors.Wrapf(err, "test DB file %q - FIX: %s", cfg.Options.DBPath, _readmeURL)
	}
	var err error
	var exportPathAbs string
	if exportPathAbs, err = filepath.Abs(cfg.Options.ExportPath); err != nil {
		return errors.Wrapf(err, "convert export path %q to an absolute path", cfg.Options.ExportPath)
	}
	cfg.Options.ExportPath = exportPathAbs
	if ok, err := cfg.OS.FileExist(exportPathAbs); err != nil {
		return errors.Wrapf(err, "check export path %q", exportPathAbs)
	} else if ok {
		return fmt.Errorf("export folder %q already exists - FIX: move it or specify a different export path with the --export-path option", exportPathAbs)
	}
	var attPathAbs string
	if attPathAbs, err = filepath.Abs(cfg.Options.AttachmentsPath); err != nil {
		return errors.Wrapf(err, "convert attachments path %q to an absolute path", cfg.Options.AttachmentsPath)
	}
	cfg.Options.AttachmentsPath = attPathAbs
	return nil
}

func printResults(version, exportPath string, c counts, duration time.Duration) {
	log.Printf(`%sBAGOUP RESULTS:
bagoup version: %s
Invocation: %s
Export folder: %q
Export files written: %d
Chats exported: %d
Valid messages exported: %d
Invalid messages exported (see warnings above): %d
Attachments copied: %s
Attachments referenced or embedded: %s
Attachments embedded: %s
Attachments missing (see warnings above): %d
HEIC conversions completed: %d
HEIC conversions failed (see warnings above): %d
Time elapsed: %s%s`,
		"\x1b[1m",
		version,
		strings.Join(os.Args, " "),
		exportPath,
		c.files,
		c.chats,
		c.messages,
		c.messagesInvalid,
		makeAttachmentsString(c.attachmentsCopied),
		makeAttachmentsString(c.attachments),
		makeAttachmentsString(c.attachmentsEmbedded),
		c.attachmentsMissing,
		c.conversions,
		c.conversionsFailed,
		duration.String(),
		"\x1b[0m",
	)
}

func makeAttachmentsString(attCounts map[string]int) (attString string) {
	attCount := 0
	for mimeType, count := range attCounts {
		attCount += count
		attString += fmt.Sprintf("\n\t%s: %d", mimeType, count)
	}
	attString = fmt.Sprintf("%d%s", attCount, attString)
	return
}

// The tilde expansion file saves the home directory in the case that we have
// copied attachments with preserved paths. This file is used to know how to
// expand the tilde when it is used in the chat DB.
func (cfg configuration) writeTildeExpansionFile() error {
	if !cfg.Options.PreservePaths {
		return nil
	}
	homeDir := cfg.PathTools.GetHomeDir()
	f, err := cfg.OS.Create(filepath.Join(cfg.Options.ExportPath, PreservedPathDir, PreservedPathTildeExpansionFile))
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.WriteString(homeDir); err != nil {
		return err
	}
	return nil
}
