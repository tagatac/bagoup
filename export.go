// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package main

import (
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/chatdb"
)

func (cfg *configuration) exportChats(contactMap map[string]*vcard.Card) error {
	if err := getAttachmentPaths(cfg); err != nil {
		return err
	}
	chats, err := cfg.ChatDB.GetChats(contactMap)
	if err != nil {
		return errors.Wrap(err, "get chats")
	}

	for _, entityChats := range chats {
		if err := cfg.exportEntityChats(entityChats); err != nil {
			return err
		}
	}
	return nil
}

func getAttachmentPaths(cfg *configuration) error {
	attPaths, err := cfg.ChatDB.GetAttachmentPaths()
	if err != nil {
		return errors.Wrap(err, "get attachment paths")
	}
	cfg.AttachmentPaths = attPaths
	if cfg.Options.OutputPDF || cfg.Options.CopyAttachments {
		for _, msgPaths := range attPaths {
			if len(msgPaths) == 0 {
				continue
			}
			if msgPaths[0].Filename == "" {
				continue
			}
			if err := cfg.OS.FileAccess(msgPaths[0].Filename); err != nil {
				return errors.Wrapf(err, "access to attachments - FIX: %s", _readmeURL)
			}
			break
		}
	}
	return nil
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
		cfg.Counts.chats += 1
	}
	if mergeChats {
		if err := cfg.writeFile(entityChats.Name, guids, entityMessageIDs); err != nil {
			return err
		}
	}
	return nil
}
