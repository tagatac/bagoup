// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

// Package opsys provides an interface OS for interacting with the running
// operating system, both with the filesystem and with Mac OS commands.
package opsys

import (
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
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
)

//go:generate mockgen -destination=mock_opsys/mock_opsys.go github.com/tagatac/bagoup/opsys OS

type (
	// OS interacts with the local filesystem and operating system.
	OS interface {
		// FileExist checks if the given path already exists.
		FileExist(path string) (bool, error)
		// GetMacOSVersion checks the version of the current operating system,
		// assuming it is Mac OS.
		GetMacOSVersion() (*semver.Version, error)
		// GetContactMap gets a map of vcards indexed by phone numbers and email
		// addresses specified in those cards, from the vcard file at the given
		// path.
		GetContactMap(path string) (map[string]*vcard.Card, error)
		// ExportChats writes all of the chats from the given ChatDB into searchable
		// formatted text files in the given export path.
		ExportChats(
			cdb chatdb.ChatDB,
			exportPath string,
			contactMap map[string]*vcard.Card,
			handleMap map[int]string,
			macOSVersion *semver.Version,
		) error
	}

	opSys struct {
		afero.Fs
		osStat      func(string) (os.FileInfo, error)
		execCommand func(string, ...string) *exec.Cmd
	}
)

// NewOS returns an OS from a given filesystem, os Stat, and exec Command.
func NewOS(fs afero.Fs, osStat func(string) (os.FileInfo, error), execCommand func(string, ...string) *exec.Cmd) OS {
	return opSys{Fs: fs, osStat: osStat, execCommand: execCommand}
}

func (s opSys) FileExist(path string) (bool, error) {
	_, err := s.osStat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, errors.Wrapf(err, "check existence of file %q", path)
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
	f, err := s.Fs.Open(contactsFilePath)
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

func (s opSys) ExportChats(
	cdb chatdb.ChatDB,
	exportPath string,
	contactMap map[string]*vcard.Card,
	handleMap map[int]string,
	macOSVersion *semver.Version,
) error {
	chats, err := cdb.GetChats(contactMap)
	if err != nil {
		return errors.Wrap(err, "get chats")
	}
	for _, chat := range chats {
		chatDirPath := path.Join(exportPath, chat.DisplayName)
		if err := s.Fs.MkdirAll(chatDirPath, os.ModePerm); err != nil {
			return errors.Wrapf(err, "create directory %q", chatDirPath)
		}
		chatPath := path.Join(chatDirPath, fmt.Sprintf("%s.txt", chat.GUID))
		chatFile, err := s.Fs.OpenFile(chatPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return errors.Wrapf(err, "open/create file %s", chatPath)
		}
		defer chatFile.Close()

		messageIDs, err := cdb.GetMessageIDs(chat.ID)
		if err != nil {
			return errors.Wrapf(err, "get message IDs for chat ID %d", chat.ID)
		}
		for _, messageID := range messageIDs {
			msg, err := cdb.GetMessage(messageID, handleMap, macOSVersion)
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
