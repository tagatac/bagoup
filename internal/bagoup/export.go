// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package bagoup

import (
	"path/filepath"

	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/chatdb"
)

type countsAndError struct {
	cnts *counts
	err  error
}

func (cfg *configuration) exportChats(contactMap map[string]*vcard.Card) ([]*counts, error) {
	if err := getAttachmentPaths(cfg); err != nil {
		return []*counts{}, err
	}
	chats, err := cfg.ChatDB.GetChats(contactMap)
	if err != nil {
		return []*counts{}, errors.Wrap(err, "get chats")
	}

	results := make(chan countsAndError)
	for _, entityChats := range chats {
		go func(ec chatdb.EntityChats) {
			cnts, err := cfg.exportEntityChats(ec)
			results <- countsAndError{
				cnts: cnts,
				err:  err,
			}
		}(entityChats)
	}

	allCounts := []*counts{}
	for result := range results {
		if result.err != nil {
			return allCounts, result.err
		}
		allCounts = append(allCounts, result.cnts)
		if len(allCounts) == len(chats) {
			break
		}
	}
	return allCounts, nil
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

func (cfg *configuration) exportEntityChats(entityChats chatdb.EntityChats) (*counts, error) {
	cnts := &counts{
		attachments:         map[string]int{},
		attachmentsCopied:   map[string]int{},
		attachmentsEmbedded: map[string]int{},
	}
	mergeChats := !cfg.Options.SeparateChats
	var guids []string
	var entityMessageIDs []chatdb.DatedMessageID
	for _, chat := range entityChats.Chats {
		messageIDs, err := cfg.ChatDB.GetMessageIDs(chat.ID)
		if err != nil {
			return cnts, errors.Wrapf(err, "get message IDs for chat ID %d", chat.ID)
		}
		if mergeChats {
			guids = append(guids, chat.GUID)
			entityMessageIDs = append(entityMessageIDs, messageIDs...)
		} else {
			if err := cfg.writeFile(cnts, entityChats.Name, []string{chat.GUID}, messageIDs); err != nil {
				return cnts, err
			}
		}
		cnts.chats++
	}
	if mergeChats {
		if err := cfg.writeFile(cnts, entityChats.Name, guids, entityMessageIDs); err != nil {
			return cnts, err
		}
	}
	return cnts, nil
}
