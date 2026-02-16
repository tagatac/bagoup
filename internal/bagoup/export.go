// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package bagoup

import (
	"path/filepath"

	progressbar "github.com/elulcao/progress-bar/cmd"
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/v2/chatdb"
)

func (cfg *configuration) exportChats(contactMap map[string]*vcard.Card) error {
	if err := getAttachmentPaths(cfg); err != nil {
		return err
	}
	chats, err := cfg.ChatDB.GetChats(contactMap)
	if err != nil {
		return errors.Wrap(err, "get chats")
	}
	chats = filterEntities(cfg.Options.Entities, chats)

	bar := progressbar.NewPBar()
	bar.SignalHandler()
	bar.Total = uint16(len(chats))
	for i, entityChats := range chats {
		bar.RenderPBar(i)
		if err := cfg.exportEntityChats(entityChats); err != nil {
			return err
		}
	}
	return nil
}

func getAttachmentPaths(cfg *configuration) error {
	attPaths, err := cfg.ChatDB.GetAttachmentPaths(cfg.PathTools)
	if err != nil {
		return errors.Wrap(err, "get attachment paths")
	}
	cfg.attachmentPaths = attPaths
	if cfg.Options.OutputPDF || cfg.Options.CopyAttachments {
		for _, msgPaths := range attPaths {
			if len(msgPaths) == 0 {
				continue
			}
			if msgPaths[0].Filename == "" {
				continue
			}
			attPath := filepath.Join(cfg.Options.AttachmentsPath, msgPaths[0].Filename)
			if err := cfg.OS.FileAccess(attPath); err != nil {
				return errors.Wrapf(err, "access to attachments - FIX: %s", _readmeURL)
			}
			break
		}
	}
	return nil
}

func filterEntities(entities []string, chats []chatdb.EntityChats) []chatdb.EntityChats {
	if len(entities) == 0 {
		return chats
	}
	result := []chatdb.EntityChats{}
	for _, entityChats := range chats {
		for _, entity := range entities {
			if entityChats.Name == entity {
				result = append(result, entityChats)
			}
		}
	}
	return result
}

func (cfg *configuration) exportEntityChats(entityChats chatdb.EntityChats) error {
	mergeChats := !cfg.Options.SeparateChats
	var guids []string
	var entityMessageIDs []chatdb.DatedMessageID
	for _, chat := range entityChats.Chats {
		messageIDs, err := cfg.ChatDB.GetMessageIDs(chat.ID)
		if err != nil {
			return errors.Wrapf(err, "get message IDs for chat ID %d", chat.ID)
		}
		if mergeChats {
			guids = append(guids, chat.GUID)
			entityMessageIDs = append(entityMessageIDs, messageIDs...)
		} else {
			if err := cfg.writeFile(entityChats.Name, []string{chat.GUID}, messageIDs); err != nil {
				return err
			}
		}
		cfg.counts.chats++
	}
	if mergeChats {
		if err := cfg.writeFile(entityChats.Name, guids, entityMessageIDs); err != nil {
			return err
		}
	}
	return nil
}
