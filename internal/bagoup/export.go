// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package bagoup

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	progressbar "github.com/elulcao/progress-bar/cmd"
	"github.com/emersion/go-vcard"
	"github.com/tagatac/bagoup/v2/chatdb"
)

func (cfg *configuration) exportChats(contactMap map[string]*vcard.Card) error {
	if err := getAttachmentPaths(cfg); err != nil {
		return err
	}
	chats, err := cfg.ChatDB.GetChats(contactMap)
	if err != nil {
		return fmt.Errorf("get chats: %w", err)
	}
	chats = filterEntities(cfg.Options.Entities, chats)

	workers := max(1, runtime.NumCPU()-1)
	jobs := make(chan chatdb.EntityChats, len(chats))
	type result struct {
		counts counts
		err    error
	}
	results := make(chan result, len(chats))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ec := range jobs {
				localCfg := *cfg
				localCfg.counts = newCounts()
				err := localCfg.exportEntityChats(ec)
				results <- result{localCfg.counts, err}
			}
		}()
	}

	bar := progressbar.NewPBar()
	bar.SignalHandler()
	bar.Total = uint16(len(chats))
	for _, ec := range chats {
		jobs <- ec
	}
	close(jobs)

	// Collect results for bar updates. mergeCounts must not run concurrently
	// with localCfg := *cfg in workers (both access cfg.counts memory).
	// Draining all len(chats) results guarantees no more jobs remain;
	// wg.Wait() guarantees workers have exited before any cfg write.
	collected := make([]result, 0, len(chats))
	for i := range chats {
		r := <-results
		bar.RenderPBar(i)
		collected = append(collected, r)
	}
	wg.Wait()

	var firstErr error
	for _, r := range collected {
		cfg.mergeCounts(r.counts)
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
	}
	return firstErr
}

func getAttachmentPaths(cfg *configuration) error {
	attPaths, err := cfg.ChatDB.GetAttachmentPaths(cfg.PathTools)
	if err != nil {
		return fmt.Errorf("get attachment paths: %w", err)
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
				return fmt.Errorf("access to attachments - FIX: %s: %w", _readmeURL, err)
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
			return fmt.Errorf("get message IDs for chat ID %d: %w", chat.ID, err)
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
