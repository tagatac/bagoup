// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package bagoup

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/tagatac/bagoup/v2/chatdb"
	"github.com/tagatac/bagoup/v2/opsys"
	"github.com/tagatac/bagoup/v2/opsys/pdfgen"
)

var openFilesLimitMu sync.Mutex

const (
	_filenamePrefixMaxLength = 251
	_pdfPreferredMessages    = 2048
	_pdfMaxMessages          = 3072
)

type writeJob struct {
	entityName string
	chatPath   string
	messageIDs []chatdb.DatedMessageID
	attDir     string
}

func (cfg *configuration) prepareFileJobs(entityName string, guids []string, messageIDs []chatdb.DatedMessageID) ([]writeJob, error) {
	chatDirPath := filepath.Join(cfg.Options.ExportPath, entityName)
	if err := cfg.OS.MkdirAll(chatDirPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create directory %q: %w", chatDirPath, err)
	}
	filename := strings.Join(guids, ";;;")
	if len(filename) > _filenamePrefixMaxLength {
		filename = filename[:_filenamePrefixMaxLength-1]
	}
	chatPathNoExt := filepath.Join(chatDirPath, filename)
	attDir := filepath.Join(chatDirPath, "attachments")
	if cfg.Options.CopyAttachments && !cfg.Options.PreservePaths {
		if err := cfg.OS.MkdirAll(attDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("create directory %q: %w", attDir, err)
		}
	}
	sort.SliceStable(messageIDs, func(i, j int) bool { return messageIDs[i].Date < messageIDs[j].Date })
	if !cfg.Options.OutputPDF {
		return []writeJob{{entityName, chatPathNoExt + ".txt", messageIDs, attDir}}, nil
	}
	var jobs []writeJob
	fileIdx := 1
	var msgIdx int
	for msgIdx = 0; len(messageIDs)-_pdfMaxMessages > msgIdx; msgIdx += _pdfPreferredMessages {
		jobs = append(jobs, writeJob{
			entityName: entityName,
			chatPath:   fmt.Sprintf("%s.%d.pdf", chatPathNoExt, fileIdx),
			messageIDs: messageIDs[msgIdx : msgIdx+_pdfPreferredMessages],
			attDir:     attDir,
		})
		fileIdx++
	}
	lastPath := chatPathNoExt + ".pdf"
	if fileIdx > 1 {
		lastPath = fmt.Sprintf("%s.%d.pdf", chatPathNoExt, fileIdx)
	}
	return append(jobs, writeJob{entityName, lastPath, messageIDs[msgIdx:], attDir}), nil
}

func (cfg *configuration) writeChunk(job writeJob) error {
	chatFile, err := cfg.OS.Create(job.chatPath)
	if err != nil {
		return fmt.Errorf("create file %q: %w", job.chatPath, err)
	}
	defer chatFile.Close()
	var outFile opsys.OutFile
	if cfg.Options.OutputPDF {
		if cfg.Options.UseWkhtmltopdf {
			pdfg, err := pdfgen.NewPDFGenerator(chatFile)
			if err != nil {
				return fmt.Errorf("create PDF generator: %w", err)
			}
			outFile = cfg.OS.NewWkhtmltopdfFile(job.entityName, chatFile, pdfg, cfg.Options.IncludePPA)
		} else {
			outFile = cfg.OS.NewWeasyPrintFile(job.entityName, chatFile, cfg.Options.IncludePPA)
		}
	} else {
		outFile = cfg.OS.NewTxtOutFile(chatFile)
	}
	return cfg.handleFileContents(outFile, job.messageIDs, job.attDir)
}

func (cfg *configuration) handleFileContents(outFile opsys.OutFile, messageIDs []chatdb.DatedMessageID, attDir string) error {
	msgCount, invalidCount := 0, 0
	for _, messageID := range messageIDs {
		msg, ok, err := cfg.ChatDB.GetMessage(messageID.ID, cfg.handleMap)
		if err != nil {
			return fmt.Errorf("get message with ID %d: %w", messageID.ID, err)
		}
		if err := outFile.WriteMessage(msg); err != nil {
			return fmt.Errorf("write message %q to file %q: %w", msg, outFile.Name(), err)
		}
		if err := cfg.handleAttachments(outFile, messageID.ID, attDir); err != nil {
			return fmt.Errorf("chat file %q - message %d: %w", outFile.Name(), messageID.ID, err)
		}
		if ok {
			msgCount++
		} else {
			invalidCount++
		}
	}
	imgCount, err := outFile.Stage()
	if err != nil {
		return fmt.Errorf("stage chat file %q for writing: %w", outFile.Name(), err)
	}
	if err := cfg.ensureOpenFilesLimit(imgCount, outFile); err != nil {
		return err
	}
	if err := outFile.Flush(); err != nil {
		return fmt.Errorf("flush chat file %q to disk: %w", outFile.Name(), err)
	}
	cfg.counts.files++
	cfg.counts.messages += msgCount
	cfg.counts.messagesInvalid += invalidCount
	return nil
}

func (cfg *configuration) ensureOpenFilesLimit(imgCount int, outFile opsys.OutFile) error {
	openFilesLimitMu.Lock()
	defer openFilesLimitMu.Unlock()
	openFilesLimit, err := cfg.OS.GetOpenFilesLimit()
	if err != nil {
		return err
	}
	if imgCount*2 > openFilesLimit {
		if err := cfg.OS.SetOpenFilesLimit(imgCount * 2); err != nil {
			return fmt.Errorf("chat file %q - increase the open file limit from %d to %d to support %d embedded images: %w", outFile.Name(), openFilesLimit, imgCount*2, imgCount, err)
		}
	}
	return nil
}

func (cfg *configuration) handleAttachments(outFile opsys.OutFile, msgID int, attDir string) error {
	msgPaths, ok := cfg.attachmentPaths[msgID]
	if !ok {
		return nil
	}
	for _, att := range msgPaths {
		att.Filepath = filepath.Join(cfg.Options.AttachmentsPath, att.Filename)
		err := cfg.validateAttachmentPath(att)
		if _, ok := err.(errorMissingAttachment); ok {
			// Attachment is missing. Just reference it, and skip copying/embedding.
			cfg.counts.attachmentsMissing++
			slog.Warn(err.Error(),
				"chat file", outFile.Name(),
				"message ID", msgID,
				slog.Group("attachment",
					"type", att.MIMEType,
					"name", att.TransferName,
					"ID", att.ID,
				))
			if err := outFile.ReferenceAttachment(att.TransferName); err != nil {
				return fmt.Errorf("reference attachment %q: %w", att.TransferName, err)
			}
			cfg.counts.attachments[att.MIMEType]++
			continue
		} else if err != nil {
			return err
		}
		if err := cfg.copyAttachment(&att, attDir); err != nil {
			return err
		}
		if err := cfg.writeAttachment(outFile, att); err != nil {
			return err
		}
	}
	return nil
}

type errorMissingAttachment struct{ err error }

func (e errorMissingAttachment) Error() string { return e.err.Error() }

func (cfg configuration) validateAttachmentPath(att chatdb.Attachment) error {
	if att.Filename == "" {
		return errorMissingAttachment{err: errors.New("attachment has no local filename")}
	}
	if ok, err := cfg.OS.FileExist(att.Filepath); err != nil {
		return fmt.Errorf("check existence of file %q - POSSIBLE FIX: %s: %w", att.Filepath, _readmeURL, err)
	} else if !ok {
		return errorMissingAttachment{err: errors.New("attachment does not exist locally")}
	}
	return nil
}

func (cfg *configuration) copyAttachment(att *chatdb.Attachment, attDir string) error {
	if !cfg.Options.CopyAttachments {
		return nil
	}
	unique := true
	if cfg.Options.PreservePaths {
		unique = false
		attDir = filepath.Join(cfg.Options.ExportPath, PreservedPathDir, filepath.Dir(att.Filename))
		if err := cfg.OS.MkdirAll(attDir, os.ModePerm); err != nil {
			return fmt.Errorf("create directory %q: %w", attDir, err)
		}
	}
	dstPath, err := cfg.OS.CopyFile(att.Filepath, attDir, unique)
	if err != nil {
		return fmt.Errorf("copy attachment %q to %q: %w", att.Filepath, attDir, err)
	}
	att.Filepath = dstPath
	cfg.counts.attachmentsCopied[att.MIMEType]++
	return nil
}

func (cfg *configuration) writeAttachment(outFile opsys.OutFile, att chatdb.Attachment) error {
	attPath, mimeType := att.Filepath, att.MIMEType
	if cfg.Options.OutputPDF {
		if jpgPath, err := cfg.ImgConverter.ConvertHEIC(attPath); err != nil {
			cfg.counts.conversionsFailed++
			slog.Warn("failed to convert HEIC file to JPEG",
				"err", err,
				"chat file", outFile.Name(),
				"HEIC file", attPath,
			)
		} else if jpgPath != attPath {
			cfg.counts.conversions++
			attPath, mimeType = jpgPath, "image/jpeg"
		}
	}
	embedded, err := outFile.WriteAttachment(attPath)
	if err != nil {
		return fmt.Errorf("include attachment %q: %w", attPath, err)
	}
	if embedded {
		cfg.counts.attachmentsEmbedded[mimeType]++
	}
	cfg.counts.attachments[mimeType]++
	return nil
}
