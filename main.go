// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See the LICENSE file for full usage terms.

package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"unicode"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
)

const (
	_dbFileName       = "chat.db"
	_contactsFileName = "contacts.vcf"
	_exportFolder     = "backup"
)

var _reqOSVersion = semver.MustParse("10.13")

func main() {
	macOSVersion, err := getMacOSVersion(exec.Command)
	if err != nil {
		log.Print(errors.Wrap(err, "failed to get Mac OS version - assuming database was copied from Mac OS 10.13 or later"))
	}

	wd, err := os.Getwd()
	exitOnError("get working directory", err)
	dbPath := path.Join(wd, _dbFileName)
	exportPath := path.Join(wd, _exportFolder)
	contactsFilePath := path.Join(wd, _contactsFileName)

	db, err := sql.Open("sqlite3", dbPath)
	exitOnError("open DB file %q", err)
	defer db.Close()
	contactMap, err := getContactMap(contactsFilePath, afero.NewOsFs())
	exitOnError(fmt.Sprintf("get contacts from vcard file %q", _contactsFileName), err)
	cdb, err := chatdb.NewChatDB(db, contactMap, macOSVersion)
	exitOnError("create ChatDB", err)

	exitOnError("export chats", exportChats(cdb, exportPath, afero.NewOsFs()))
}

func exitOnError(activity string, err error) {
	if err != nil {
		log.Fatal(errors.Wrap(err, activity))
	}
}

func getMacOSVersion(execCommand func(string, ...string) *exec.Cmd) (*semver.Version, error) {
	cmd := execCommand("sw_vers", "-productVersion")
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

func getContactMap(contactsFilePath string, fs afero.Fs) (map[string]*vcard.Card, error) {
	f, err := fs.Open(contactsFilePath)
	if os.IsNotExist(err) {
		log.Print(errors.Wrapf(err, "open file %q - continuing without contacts", contactsFilePath))
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "open file %q", contactsFilePath)
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
				log.Printf("multiple contacts %q and %q share the same phone or email %q", c.PreferredValue(vcard.FieldFormattedName), card.PreferredValue(vcard.FieldFormattedName), phoneOrEmail)
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

func exportChats(cdb chatdb.ChatDB, exportPath string, fs afero.Fs) error {
	chats, err := cdb.GetChats()
	if err != nil {
		return errors.Wrap(err, "get chats")
	}
	for _, chat := range chats {
		chatDirPath := path.Join(exportPath, chat.DisplayName)
		if err := fs.MkdirAll(chatDirPath, os.ModePerm); err != nil {
			return errors.Wrapf(err, "create directory %q", chatDirPath)
		}
		chatPath := path.Join(chatDirPath, fmt.Sprintf("%s.txt", chat.GUID))
		chatFile, err := fs.OpenFile(chatPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return errors.Wrapf(err, "open/create file %s", chatPath)
		}
		defer chatFile.Close()

		messageIDs, err := cdb.GetMessageIDs(chat.ID)
		if err != nil {
			return errors.Wrapf(err, "get message IDs for chat ID %d", chat.ID)
		}
		for _, messageID := range messageIDs {
			msg, err := cdb.GetMessage(messageID)
			if err != nil {
				return errors.Wrapf(err, "get message with ID %d", messageID)
			}
			if _, err := chatFile.WriteString(msg); err != nil {
				return errors.Wrapf(err, "write message %q to file %q", msg, chatFile.Name())
			}
		}
		chatFile.Close()
	}
	return nil
}
