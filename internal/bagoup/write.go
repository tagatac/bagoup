// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package bagoup

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/opsys"
	"github.com/tagatac/bagoup/opsys/pdfgen"
)

const _filenamePrefixMaxLength = 251

func (cfg *configuration) writeFile(entityName string, guids []string, messageIDs []chatdb.DatedMessageID) error {
	chatDirPath := strings.TrimRight(filepath.Join(cfg.Options.ExportPath, entityName), ". ")
	if err := cfg.OS.MkdirAll(chatDirPath, os.ModePerm); err != nil {
		return errors.Wrapf(err, "create directory %q", chatDirPath)
	}
	filename := strings.Join(guids, ";;;")
	if len(filename) > _filenamePrefixMaxLength {
		filename = filename[:_filenamePrefixMaxLength-1]
	}
	chatPath := filepath.Join(chatDirPath, filename)
	var outFile opsys.OutFile
	if cfg.Options.OutputPDF {
		chatPath += ".pdf"
		chatFile, err := cfg.OS.Create(chatPath)
		if err != nil {
			return errors.Wrapf(err, "create file %q", chatPath)
		}
		defer chatFile.Close()
		pdfg, err := pdfgen.NewPDFGenerator(chatFile)
		if err != nil {
			return errors.Wrap(err, "create PDF generator")
		}
		outFile = cfg.OS.NewPDFOutFile(chatFile, pdfg, cfg.Options.IncludePPA)
	} else {
		chatPath += ".txt"
		chatFile, err := cfg.OS.Create(chatPath)
		if err != nil {
			return errors.Wrapf(err, "create file %q", chatPath)
		}
		defer chatFile.Close()
		outFile = cfg.OS.NewTxtOutFile(chatFile)
	}
	attDir := filepath.Join(chatDirPath, "attachments")
	if cfg.Options.CopyAttachments && !cfg.Options.PreservePaths {
		if err := cfg.OS.MkdirAll(attDir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "create directory %q", attDir)
		}
	}
	return cfg.handleFileContents(outFile, messageIDs, attDir)
}

func (cfg *configuration) handleFileContents(outFile opsys.OutFile, messageIDs []chatdb.DatedMessageID, attDir string) error {
	sort.SliceStable(messageIDs, func(i, j int) bool { return messageIDs[i].Date < messageIDs[j].Date })
	msgCount := 0
	for _, messageID := range messageIDs {
		msg, err := cfg.ChatDB.GetMessage(messageID.ID, cfg.handleMap)
		if err != nil {
			return errors.Wrapf(err, "get message with ID %d", messageID.ID)
		}
		if err := outFile.WriteMessage(msg); err != nil {
			return errors.Wrapf(err, "write message %q to file %q", msg, outFile.Name())
		}
		if err := cfg.handleAttachments(outFile, messageID.ID, attDir); err != nil {
			return errors.Wrapf(err, "chat file %q - message %d", outFile.Name(), messageID.ID)
		}
		msgCount += 1
	}
	imgCount, err := outFile.Flush()
	if err != nil {
		return errors.Wrapf(err, "flush chat file %q to disk", outFile.Name())
	}
	if openFilesLimit := cfg.OS.GetOpenFilesLimit(); imgCount*2 > openFilesLimit {
		if err := cfg.OS.SetOpenFilesLimit(imgCount * 2); err != nil {
			return errors.Wrapf(err, "chat file %q - increase the open file limit from %d to %d to support %d embedded images", outFile.Name(), openFilesLimit, imgCount*2, imgCount)
		}
	}
	cfg.counts.files += 1
	cfg.counts.messages += msgCount
	return nil
}

func (cfg *configuration) handleAttachments(outFile opsys.OutFile, msgID int, attDir string) error {
	msgPaths, ok := cfg.attachmentPaths[msgID]
	if !ok {
		return nil
	}
	for _, att := range msgPaths {
		attPath, mimeType, transferName := att.Filename, att.MIMEType, att.TransferName
		err := cfg.validateAttachmentPath(attPath)
		if _, ok := err.(errorMissingAttachment); ok {
			// Attachment is missing. Just reference it, and skip copying/embedding.
			cfg.counts.attachmentsMissing += 1
			log.Printf("WARN: chat file %q - message %d - %s attachment %q (ID %d) - %s", outFile.Name(), msgID, mimeType, transferName, att.ID, err)
			if err := outFile.ReferenceAttachment(transferName); err != nil {
				return errors.Wrapf(err, "reference attachment %q", transferName)
			}
			cfg.counts.attachments[mimeType] += 1
			continue
		} else if err != nil {
			return err
		}
		if err := cfg.copyAttachment(att, attDir); err != nil {
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

func (cfg configuration) validateAttachmentPath(attPath string) error {
	if attPath == "" {
		return errorMissingAttachment{err: errors.New("attachment has no local filename")}
	}
	attPath = filepath.Join(cfg.Options.AttachmentsPath, attPath)
	if ok, err := cfg.OS.FileExist(attPath); err != nil {
		return errors.Wrapf(err, "check existence of file %q - POSSIBLE FIX: %s", attPath, _readmeURL)
	} else if !ok {
		return errorMissingAttachment{err: errors.New("attachment does not exist locally")}
	}
	return nil
}

func (cfg *configuration) copyAttachment(att chatdb.Attachment, attDir string) error {
	if !cfg.Options.CopyAttachments {
		return nil
	}
	attPath, mimeType := att.Filename, att.MIMEType
	unique := true
	if cfg.Options.PreservePaths {
		unique = false
		attDir = filepath.Join(cfg.Options.ExportPath, "bagoup-attachments", filepath.Dir(attPath))
		if err := cfg.OS.MkdirAll(attDir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "create directory %q", attDir)
		}
	}
	attPath = filepath.Join(cfg.Options.AttachmentsPath, attPath)
	if err := cfg.OS.CopyFile(attPath, attDir, unique); err != nil {
		return errors.Wrapf(err, "copy attachment %q to %q", attPath, attDir)
	}
	cfg.counts.attachmentsCopied[mimeType] += 1
	return nil
}

func (cfg *configuration) writeAttachment(outFile opsys.OutFile, att chatdb.Attachment) error {
	attPath, mimeType := filepath.Join(cfg.Options.AttachmentsPath, att.Filename), att.MIMEType
	if cfg.Options.OutputPDF {
		if jpgPath, err := cfg.OS.HEIC2JPG(attPath); err != nil {
			cfg.counts.conversionsFailed += 1
			log.Printf("WARN: chat file %q - convert HEIC file %q to JPG: %s", outFile.Name(), attPath, err)
		} else if jpgPath != attPath {
			cfg.counts.conversions += 1
			attPath, mimeType = jpgPath, "image/jpeg"
		}
	}
	embedded, err := outFile.WriteAttachment(attPath)
	if err != nil {
		return errors.Wrapf(err, "include attachment %q", attPath)
	}
	if embedded {
		cfg.counts.attachmentsEmbedded[mimeType] += 1
	}
	cfg.counts.attachments[mimeType] += 1
	return nil
}
