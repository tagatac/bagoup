// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

// Package bagoup reads data from a macOS messsages chat database and exports
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

	"github.com/Masterminds/semver/v3"
	"github.com/emersion/go-vcard"
	"github.com/tagatac/bagoup/v2/chatdb"
	"github.com/tagatac/bagoup/v2/imgconv"
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
		imgconv.ImgConverter
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
func NewConfiguration(
	opts Options,
	s opsys.OS,
	cdb chatdb.ChatDB,
	ptools pathtools.PathTools,
	logDir string,
	startTime time.Time,
	version string,
) (Configuration, error) {
	if opts.AttachmentsPath != "/" {
		tef := filepath.Join(opts.AttachmentsPath, PreservedPathTildeExpansionFile)
		homeDir, err := s.ReadFile(tef)
		if err != nil {
			return nil, fmt.Errorf("read tilde expansion file %q - POSSIBLE FIX: create a file .tildeexpansion with the expanded home directory from the previous run and place it at the root of the preserved-paths copied attachments directory (usually %q): %w", tef, PreservedPathDir, err)
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
		return fmt.Errorf("make log directory: %w", err)
	}
	logFile, err := cfg.OS.Create(filepath.Join(cfg.logDir, "out.log"))
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()
	w := log.Writer()
	log.SetOutput(io.MultiWriter(logFile, w))
	defer log.SetOutput(w)

	if cfg.Options.MacOSVersion != nil {
		cfg.macOSVersion, err = semver.NewVersion(*cfg.Options.MacOSVersion)
		if err != nil {
			return fmt.Errorf("parse macOS version %q: %w", *cfg.Options.MacOSVersion, err)
		}
	} else if cfg.macOSVersion, err = cfg.OS.GetMacOSVersion(); err != nil {
		return fmt.Errorf("get macOS version - FIX: specify the macOS version from which chat.db was copied with the --mac-os-version option: %w", err)
	}

	var contactMap map[string]*vcard.Card
	if cfg.Options.ContactsPath != nil {
		contactMap, err = cfg.OS.GetContactMap(*cfg.Options.ContactsPath)
		if err != nil {
			return fmt.Errorf("get contacts from vcard file %q: %w", *cfg.Options.ContactsPath, err)
		}
	}

	if err := cfg.ChatDB.Init(cfg.macOSVersion); err != nil {
		return fmt.Errorf("initialize the database for reading on macOS version %s: %w", cfg.macOSVersion.String(), err)
	}

	cfg.handleMap, err = cfg.ChatDB.GetHandleMap(contactMap)
	if err != nil {
		return fmt.Errorf("get handle map: %w", err)
	}

	if cfg.Options.OutputPDF {
		tempDir, err := cfg.OS.GetTempDir()
		if err != nil {
			return fmt.Errorf("get temporary directory: %w", err)
		}
		defer cfg.OS.RmTempDir()
		cfg.ImgConverter = imgconv.NewImgConverter(tempDir)
	}

	err = cfg.exportChats(contactMap)
	printResults(cfg.version, cfg.Options.ExportPath, cfg.counts, time.Since(cfg.startTime))
	if err != nil {
		return fmt.Errorf("export chats: %w", err)
	}
	if err = cfg.writeTildeExpansionFile(); err != nil {
		return fmt.Errorf("write out tilde expansion file: %w", err)
	}
	return cfg.OS.RmTempDir()
}

func (cfg *configuration) validatePaths() error {
	if err := cfg.OS.FileAccess(cfg.Options.DBPath); err != nil {
		return fmt.Errorf("test DB file %q - FIX: %s: %w", cfg.Options.DBPath, _readmeURL, err)
	}
	var err error
	var exportPathAbs string
	if exportPathAbs, err = filepath.Abs(cfg.Options.ExportPath); err != nil {
		return fmt.Errorf("convert export path %q to an absolute path: %w", cfg.Options.ExportPath, err)
	}
	cfg.Options.ExportPath = exportPathAbs
	if ok, err := cfg.OS.FileExist(exportPathAbs); err != nil {
		return fmt.Errorf("check export path %q: %w", exportPathAbs, err)
	} else if ok {
		return fmt.Errorf("export folder %q already exists - FIX: move it or specify a different export path with the --export-path option", exportPathAbs)
	}
	var attPathAbs string
	if attPathAbs, err = filepath.Abs(cfg.Options.AttachmentsPath); err != nil {
		return fmt.Errorf("convert attachments path %q to an absolute path: %w", cfg.Options.AttachmentsPath, err)
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
