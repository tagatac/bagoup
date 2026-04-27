// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package bagoup

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	progressbar "github.com/elulcao/progress-bar/cmd"
	"github.com/emersion/go-vcard"
	"github.com/tagatac/bagoup/v2/chatdb"
	"golang.org/x/sync/errgroup"
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

	var allJobs []writeJob
	for _, ec := range chats {
		jobs, err := cfg.prepareEntityJobs(ec)
		if err != nil {
			return err
		}
		allJobs = append(allJobs, jobs...)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return cfg.runPool(ctx, allJobs)
}

func (cfg *configuration) runPool(ctx context.Context, jobs []writeJob) error {
	if len(jobs) == 0 {
		return nil
	}
	jobsCh := make(chan writeJob, len(jobs))
	for _, job := range jobs {
		jobsCh <- job
	}

	bar := progressbar.NewPBar()
	bar.SignalHandler()
	bar.Total = uint16(len(jobs))

	allDoneCtx, cancelAllDone := context.WithCancel(ctx)
	defer cancelAllDone()

	g, gCtx := errgroup.WithContext(allDoneCtx)
	var mu sync.Mutex
	var done int
	for range max(1, runtime.NumCPU()-1) {
		g.Go(func() error {
			for {
				select {
				case job := <-jobsCh:
					c := newCounts()
					if err := cfg.writeChunk(job, c); err != nil {
						return err
					}
					mu.Lock()
					cfg.mergeCounts(c)
					done++
					bar.RenderPBar(done)
					if done == len(jobs) {
						cancelAllDone()
					}
					mu.Unlock()
				case <-gCtx.Done():
					return nil
				}
			}
		})
	}
	return g.Wait()
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

func (cfg *configuration) prepareEntityJobs(entityChats chatdb.EntityChats) ([]writeJob, error) {
	mergeChats := !cfg.Options.SeparateChats
	var guids []string
	var entityMessageIDs []chatdb.DatedMessageID
	var jobs []writeJob
	for _, chat := range entityChats.Chats {
		messageIDs, err := cfg.ChatDB.GetMessageIDs(chat.ID)
		if err != nil {
			return nil, fmt.Errorf("get message IDs for chat ID %d: %w", chat.ID, err)
		}
		if mergeChats {
			guids = append(guids, chat.GUID)
			entityMessageIDs = append(entityMessageIDs, messageIDs...)
		} else {
			chatJobs, err := cfg.prepareFileJobs(entityChats.Name, []string{chat.GUID}, messageIDs)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, chatJobs...)
		}
		cfg.counts.chats++
	}
	if mergeChats {
		chatJobs, err := cfg.prepareFileJobs(entityChats.Name, guids, entityMessageIDs)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, chatJobs...)
	}
	return jobs, nil
}
