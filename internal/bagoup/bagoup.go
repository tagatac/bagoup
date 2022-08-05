// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

// Package bagoup reads data from a Mac OS messsages chat database and exports
// it to text or PDF.
package bagoup

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/opsys"
)

const _readmeURL = "https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access"

type (
	// Options are the commandline options that can be passed to the bagoup
	// command.
	Options struct {
		DBPath          string  `short:"i" long:"db-path" description:"Path to the Messages chat database file" default:"~/Library/Messages/chat.db"`
		ExportPath      string  `short:"o" long:"export-path" description:"Path to which the Messages will be exported" default:"messages-export"`
		MacOSVersion    *string `short:"m" long:"mac-os-version" description:"Version of Mac OS, e.g. '10.15', from which the Messages chat database file was copied (not needed if bagoup is running on the same Mac)"`
		ContactsPath    *string `short:"c" long:"contacts-path" description:"Path to the contacts vCard file"`
		SelfHandle      string  `short:"s" long:"self-handle" description:"Prefix to use for for messages sent by you" default:"Me"`
		SeparateChats   bool    `long:"separate-chats" description:"Do not merge chats with the same contact (e.g. iMessage and SMS) into a single file"`
		OutputPDF       bool    `short:"p" long:"pdf" description:"Export text and images to PDF files (requires full disk access)"`
		IncludePPA      bool    `long:"include-ppa" description:"Include plugin payload attachments (e.g. link previews) in generated PDFs"`
		CopyAttachments bool    `short:"a" long:"copy-attachments" description:"Copy attachments to the same folder as the chat which included them (requires full disk access)"`
		PreservePaths   bool    `short:"r" long:"preserve-paths" description:"When copying attachments, preserve the full path instead of co-locating them with the chats which included them"`
		AttachmentsPath string  `short:"t" long:"attachments-path" description:"Root path to the attachments (useful for re-running bagoup on an export with the --preserve-paths flag)" default:"/"`
		PrintVersion    bool    `short:"v" long:"version" description:"Show the version of bagoup"`
	}
	configuration struct {
		Options
		opsys.OS
		chatdb.ChatDB
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
func NewConfiguration(opts Options, s opsys.OS, cdb chatdb.ChatDB, logDir string, startTime time.Time, version string) Configuration {
	return &configuration{
		Options: opts,
		OS:      s,
		ChatDB:  cdb,
		logDir:  logDir,
		counts: counts{
			attachments:         map[string]int{},
			attachmentsCopied:   map[string]int{},
			attachmentsEmbedded: map[string]int{},
		},
		startTime: startTime,
		version:   version,
	}
}

func (cfg configuration) Run() error {
	opts := cfg.Options
	if err := validatePaths(cfg.OS, opts); err != nil {
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

	if opts.MacOSVersion != nil {
		cfg.macOSVersion, err = semver.NewVersion(*opts.MacOSVersion)
		if err != nil {
			return errors.Wrapf(err, "parse Mac OS version %q", *opts.MacOSVersion)
		}
	} else if cfg.macOSVersion, err = cfg.OS.GetMacOSVersion(); err != nil {
		return errors.Wrap(err, "get Mac OS version - FIX: specify the Mac OS version from which chat.db was copied with the --mac-os-version option")
	}

	var contactMap map[string]*vcard.Card
	if opts.ContactsPath != nil {
		contactMap, err = cfg.OS.GetContactMap(*opts.ContactsPath)
		if err != nil {
			return errors.Wrapf(err, "get contacts from vcard file %q", *opts.ContactsPath)
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
	printResults(cfg.version, opts.ExportPath, cfg.counts, time.Since(cfg.startTime))
	if err != nil {
		return errors.Wrap(err, "export chats")
	}
	return cfg.OS.RmTempDir()
}

func validatePaths(s opsys.OS, opts Options) error {
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

func printResults(version, exportPath string, c counts, duration time.Duration) {
	attCpString := makeAttachmentsString(c.attachmentsCopied)
	attString := makeAttachmentsString(c.attachments)
	attEmbdString := makeAttachmentsString(c.attachmentsEmbedded)
	log.Printf(`%sBAGOUP RESULTS:
bagoup version: %s
Export folder: %q
Export files written: %d
Chats exported: %d
Messages exported: %d
Attachments copied: %s
Attachments referenced or embedded: %s
Attachments embedded: %s
Attachments missing (see warnings above): %d
HEIC conversions completed: %d
HEIC conversions failed (see warnings above): %d
Time elapsed: %s%s`,
		"\x1b[1m",
		version,
		exportPath,
		c.files,
		c.chats,
		c.messages,
		attCpString,
		attString,
		attEmbdString,
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
