// bagoup - An export utility for Mac OS Messages.
// Copyright (C) 2020-2022 David Tagatac <david@tagatac.net>

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/jessevdk/go-flags"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/opsys"
	"github.com/tagatac/bagoup/pathtools"
)

const _readmeURL = "https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access"

type (
	options struct {
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
	}
	configuration struct {
		Options options
		opsys.OS
		chatdb.ChatDB
		MacOSVersion    *semver.Version
		HandleMap       map[int]string
		AttachmentPaths map[int][]chatdb.Attachment
		Counts          counts
		StartTime       time.Time
	}
	counts struct {
		files               int
		chats               int
		messages            int
		attachments         map[string]int
		attachmentsEmbedded map[string]int
		attachmentsMissing  int
		conversions         int
		conversionsFailed   int
	}
)

func main() {
	startTime := time.Now()
	var opts options
	_, err := flags.Parse(&opts)
	if err != nil && err.(*flags.Error).Type == flags.ErrHelp {
		os.Exit(0)
	}
	logFatalOnErr(errors.Wrap(err, "parse flags"))
	dbPath, err := pathtools.ReplaceTilde(opts.DBPath)
	logFatalOnErr(errors.Wrap(err, "replace tilde"))
	opts.DBPath = dbPath

	s, err := opsys.NewOS(afero.NewOsFs(), os.Stat, exec.Command)
	logFatalOnErr(errors.Wrap(err, "instantiate OS"))
	db, err := sql.Open("sqlite3", opts.DBPath)
	logFatalOnErr(errors.Wrapf(err, "open DB file %q", opts.DBPath))
	defer db.Close()
	cdb := chatdb.NewChatDB(db, opts.SelfHandle)

	config := configuration{
		OS:        s,
		ChatDB:    cdb,
		Options:   opts,
		Counts:    counts{attachments: map[string]int{}, attachmentsEmbedded: map[string]int{}},
		StartTime: startTime,
	}
	logFatalOnErr(config.bagoup())
}

func logFatalOnErr(err error) {
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func (config configuration) bagoup() error {
	opts := config.Options
	if err := validatePaths(config.OS, opts); err != nil {
		return err
	}

	var err error
	if opts.MacOSVersion != nil {
		config.MacOSVersion, err = semver.NewVersion(*opts.MacOSVersion)
		if err != nil {
			return errors.Wrapf(err, "parse Mac OS version %q", *opts.MacOSVersion)
		}
	} else if config.MacOSVersion, err = config.OS.GetMacOSVersion(); err != nil {
		return errors.Wrap(err, "get Mac OS version - FIX: specify the Mac OS version from which chat.db was copied with the --mac-os-version option")
	}

	var contactMap map[string]*vcard.Card
	if opts.ContactsPath != nil {
		contactMap, err = config.OS.GetContactMap(*opts.ContactsPath)
		if err != nil {
			return errors.Wrapf(err, "get contacts from vcard file %q", *opts.ContactsPath)
		}
	}

	config.HandleMap, err = config.ChatDB.GetHandleMap(contactMap)
	if err != nil {
		return errors.Wrap(err, "get handle map")
	}

	defer config.RmTempDir()
	err = config.exportChats(contactMap)
	printResults(opts.ExportPath, config.Counts, time.Since(config.StartTime))
	if err != nil {
		return errors.Wrap(err, "export chats")
	}
	return config.RmTempDir()
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

func (config *configuration) exportChats(contactMap map[string]*vcard.Card) error {
	if err := getAttachmentPaths(config); err != nil {
		return err
	}
	chats, err := config.GetChats(contactMap)
	if err != nil {
		return errors.Wrap(err, "get chats")
	}

	for _, entityChats := range chats {
		if err := config.exportEntityChats(entityChats); err != nil {
			return err
		}
	}
	return nil
}

func getAttachmentPaths(config *configuration) error {
	attPaths, missing, err := config.GetAttachmentPaths()
	if err != nil {
		return errors.Wrap(err, "get attachment paths")
	}
	config.AttachmentPaths = attPaths
	config.Counts.attachmentsMissing = missing
	if config.Options.OutputPDF || config.Options.CopyAttachments {
		for _, msgPaths := range attPaths {
			if len(msgPaths) == 0 {
				continue
			}
			if err := config.FileAccess(msgPaths[0].Filename); err != nil {
				return errors.Wrapf(err, "access to attachments - FIX: %s", _readmeURL)
			}
			break
		}
	}
	return nil
}

func (config *configuration) exportEntityChats(entityChats chatdb.EntityChats) error {
	mergeChats := !config.Options.SeparateChats
	var guids []string
	var entityMessageIDs []chatdb.DatedMessageID
	for _, chat := range entityChats.Chats {
		messageIDs, err := config.GetMessageIDs(chat.ID)
		if err != nil {
			return errors.Wrapf(err, "get message IDs for chat ID %d", chat.ID)
		}
		if mergeChats {
			guids = append(guids, chat.GUID)
			entityMessageIDs = append(entityMessageIDs, messageIDs...)
		} else {
			if err := config.writeFile(entityChats.Name, []string{chat.GUID}, messageIDs); err != nil {
				return err
			}
		}
		config.Counts.chats += 1
	}
	if mergeChats {
		if err := config.writeFile(entityChats.Name, guids, entityMessageIDs); err != nil {
			return err
		}
	}
	return nil
}

func (config *configuration) writeFile(entityName string, guids []string, messageIDs []chatdb.DatedMessageID) error {
	chatDirPath := filepath.Join(config.Options.ExportPath, entityName)
	if err := config.MkdirAll(chatDirPath, os.ModePerm); err != nil {
		return errors.Wrapf(err, "create directory %q", chatDirPath)
	}
	filename := strings.Join(guids, ";;;")
	chatPath := filepath.Join(chatDirPath, filename)
	outFile, err := config.NewOutFile(chatPath, config.Options.OutputPDF, config.Options.IncludePPA)
	if err != nil {
		return errors.Wrapf(err, "open/create file %q", chatPath)
	}
	defer outFile.Close()
	attDir := filepath.Join(chatDirPath, "attachments")
	if config.Options.CopyAttachments && !config.Options.PreservePaths {
		if err := config.Mkdir(attDir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "create directory %q", attDir)
		}
	}

	sort.SliceStable(messageIDs, func(i, j int) bool { return messageIDs[i].Date < messageIDs[j].Date })
	msgCount := 0
	for _, messageID := range messageIDs {
		msg, err := config.GetMessage(messageID.ID, config.HandleMap, config.MacOSVersion)
		if err != nil {
			return errors.Wrapf(err, "get message with ID %d", messageID.ID)
		}
		if err := outFile.WriteMessage(msg); err != nil {
			return errors.Wrapf(err, "write message %q to file %q", msg, outFile.Name())
		}
		if err := config.copyAndWriteAttachments(outFile, messageID.ID, attDir); err != nil {
			return errors.Wrapf(err, "chat file %q - message %d", outFile.Name(), messageID.ID)
		}
		msgCount += 1
	}
	imgCount, err := outFile.Stage()
	if err != nil {
		return errors.Wrapf(err, "stage chat file %q for writing/closing", outFile.Name())
	}
	if openFilesLimit := config.GetOpenFilesLimit(); imgCount*2 > openFilesLimit {
		if err := config.SetOpenFilesLimit(imgCount * 2); err != nil {
			return errors.Wrapf(err, "chat file %q - increase the open file limit from %d to %d to support %d embedded images", outFile.Name(), openFilesLimit, imgCount*2, imgCount)
		}
	}
	if err := outFile.Close(); err != nil {
		return errors.Wrapf(err, "write/close chat file %q", outFile.Name())
	}
	config.Counts.files += 1
	config.Counts.messages += msgCount
	return nil
}

func (config *configuration) copyAndWriteAttachments(outFile opsys.OutFile, msgID int, attDir string) error {
	msgPaths, ok := config.AttachmentPaths[msgID]
	if !ok {
		return nil
	}
	for _, att := range msgPaths {
		attPath, mimeType := att.Filename, att.MIMEType
		if ok, err := config.FileExist(attPath); err != nil {
			return errors.Wrapf(err, "check existence of file %q - POSSIBLE FIX: %s", attPath, _readmeURL)
		} else if !ok {
			config.Counts.attachmentsMissing += 1
			log.Printf("WARN: chat file %q - message %d - attachment %q does not exist locally", outFile.Name(), msgID, attPath)
			continue
		}
		if config.Options.CopyAttachments {
			unique := true
			if config.Options.PreservePaths {
				unique = false
				attDir = filepath.Join(config.Options.ExportPath, "bagoup-attachments", filepath.Dir(attPath))
				if err := config.MkdirAll(attDir, os.ModePerm); err != nil {
					return errors.Wrapf(err, "create directory %q", attDir)
				}
			}
			if err := config.CopyFile(attPath, attDir, unique); err != nil {
				return errors.Wrapf(err, "copy attachment %q to %q", attPath, attDir)
			}
		}
		if config.Options.OutputPDF {
			if jpgPath, err := config.HEIC2JPG(attPath); err != nil {
				config.Counts.conversionsFailed += 1
				log.Printf("WARN: chat file %q - convert HEIC file %q to JPG: %s", outFile.Name(), attPath, err)
			} else if jpgPath != attPath {
				config.Counts.conversions += 1
				attPath, mimeType = jpgPath, "image/jpeg"
			}
		}
		embedded, err := outFile.WriteAttachment(attPath)
		if err != nil {
			return errors.Wrapf(err, "include attachment %q", attPath)
		}
		if embedded {
			config.Counts.attachmentsEmbedded[mimeType] += 1
		}
		config.Counts.attachments[mimeType] += 1
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
