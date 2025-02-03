// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package bagoup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/v2/chatdb"
	"github.com/tagatac/bagoup/v2/opsys"
	"github.com/tagatac/bagoup/v2/opsys/pdfgen"
)

const (
	_filenamePrefixMaxLength = 251
	_pdfPreferredMessages    = 2048
	_pdfMaxMessages          = 3072
)

func (cfg *configuration) writeFile(entityName string, guids []string, messageIDs []chatdb.DatedMessageID) error {
	chatDirPath := strings.TrimRight(filepath.Join(cfg.Options.ExportPath, entityName), ". ")
	if err := cfg.OS.MkdirAll(chatDirPath, os.ModePerm); err != nil {
		return errors.Wrapf(err, "create directory %q", chatDirPath)
	}
	filename := strings.Join(guids, ";;;")
	if len(filename) > _filenamePrefixMaxLength {
		filename = filename[:_filenamePrefixMaxLength-1]
	}
	chatPathNoExt := filepath.Join(chatDirPath, filename)
	attDir := filepath.Join(chatDirPath, "attachments")
	if cfg.Options.CopyAttachments && !cfg.Options.PreservePaths {
		if err := cfg.OS.MkdirAll(attDir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "create directory %q", attDir)
		}
	}
	sort.SliceStable(messageIDs, func(i, j int) bool { return messageIDs[i].Date < messageIDs[j].Date })
	if cfg.Options.OutputPDF {
		return cfg.writePDFs(messageIDs, chatPathNoExt, attDir)
	}
	return cfg.writeTxt(messageIDs, chatPathNoExt, attDir)
}

func (cfg *configuration) writeTxt(messageIDs []chatdb.DatedMessageID, chatPathNoExt, attDir string) error {
	chatPath := chatPathNoExt + ".txt"
	chatFile, err := cfg.OS.Create(chatPath)
	if err != nil {
		return errors.Wrapf(err, "create file %q", chatPath)
	}
	defer chatFile.Close()
	outFile := cfg.OS.NewTxtOutFile(chatFile)
	return cfg.handleFileContents(outFile, messageIDs, attDir)
}

func (cfg *configuration) writePDFs(messageIDs []chatdb.DatedMessageID, chatPathNoExt, attDir string) error {
	type messageIDsAndChatPath struct {
		messageIDs []chatdb.DatedMessageID
		chatPath   string
	}
	idsAndPaths := []messageIDsAndChatPath{}
	fileIdx := 1
	var msgIdx int
	for msgIdx = 0; len(messageIDs)-_pdfMaxMessages > msgIdx; msgIdx += _pdfPreferredMessages {
		idsAndPaths = append(idsAndPaths, messageIDsAndChatPath{
			messageIDs: messageIDs[msgIdx : msgIdx+_pdfPreferredMessages],
			chatPath:   fmt.Sprintf("%s.%d.pdf", chatPathNoExt, fileIdx),
		})
		fileIdx++
	}
	lastChatPath := chatPathNoExt + ".pdf"
	if fileIdx > 1 {
		lastChatPath = fmt.Sprintf("%s.%d.pdf", chatPathNoExt, fileIdx)
	}
	idsAndPaths = append(idsAndPaths, messageIDsAndChatPath{
		messageIDs: messageIDs[msgIdx:],
		chatPath:   lastChatPath,
	})

	for _, idsAndPath := range idsAndPaths {
		chatPath := idsAndPath.chatPath
		chatFile, err := cfg.OS.Create(chatPath)
		if err != nil {
			return errors.Wrapf(err, "create file %q", chatPath)
		}
		defer chatFile.Close()
		var outFile opsys.OutFile
		if cfg.Options.UseWkhtmltopdf {
			pdfg, err := pdfgen.NewPDFGenerator(chatFile)
			if err != nil {
				return errors.Wrap(err, "create PDF generator")
			}
			outFile = cfg.OS.NewWkhtmltopdfFile(chatFile, pdfg, cfg.Options.IncludePPA)
		} else {
			outFile = cfg.OS.NewWeasyPrintFile(chatFile, cfg.Options.IncludePPA)
		}
		if err := cfg.handleFileContents(outFile, idsAndPath.messageIDs, attDir); err != nil {
			return err
		}
	}
	return nil
}

func (cfg *configuration) handleFileContents(outFile opsys.OutFile, messageIDs []chatdb.DatedMessageID, attDir string) error {
	msgCount, invalidCount := 0, 0
	for _, messageID := range messageIDs {
		msg, ok, err := cfg.ChatDB.GetMessage(messageID.ID, cfg.handleMap)
		if err != nil {
			return errors.Wrapf(err, "get message with ID %d", messageID.ID)
		}
		if err := outFile.WriteMessage(msg); err != nil {
			return errors.Wrapf(err, "write message %q to file %q", msg, outFile.Name())
		}
		if err := cfg.handleAttachments(outFile, messageID.ID, attDir); err != nil {
			return errors.Wrapf(err, "chat file %q - message %d", outFile.Name(), messageID.ID)
		}
		if ok {
			msgCount++
		} else {
			invalidCount++
		}
	}
	imgCount, err := outFile.Stage()
	if err != nil {
		return errors.Wrapf(err, "stage chat file %q for writing", outFile.Name())
	}
	openFilesLimit, err := cfg.OS.GetOpenFilesLimit()
	if err != nil {
		return err
	}
	if imgCount*2 > openFilesLimit {
		if err := cfg.OS.SetOpenFilesLimit(imgCount * 2); err != nil {
			return errors.Wrapf(err, "chat file %q - increase the open file limit from %d to %d to support %d embedded images", outFile.Name(), openFilesLimit, imgCount*2, imgCount)
		}
	}
	if err := outFile.Flush(); err != nil {
		return errors.Wrapf(err, "flush chat file %q to disk", outFile.Name())
	}
	cfg.counts.files++
	cfg.counts.messages += msgCount
	cfg.counts.messagesInvalid += invalidCount
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
			log.Printf("WARN: chat file %q - message %d - %s attachment %q (ID %d) - %s", outFile.Name(), msgID, att.MIMEType, att.TransferName, att.ID, err)
			if err := outFile.ReferenceAttachment(att.TransferName); err != nil {
				return errors.Wrapf(err, "reference attachment %q", att.TransferName)
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
		return errors.Wrapf(err, "check existence of file %q - POSSIBLE FIX: %s", att.Filepath, _readmeURL)
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
			return errors.Wrapf(err, "create directory %q", attDir)
		}
	}
	dstPath, err := cfg.OS.CopyFile(att.Filepath, attDir, unique)
	if err != nil {
		return errors.Wrapf(err, "copy attachment %q to %q", att.Filepath, attDir)
	}
	att.Filepath = dstPath
	cfg.counts.attachmentsCopied[att.MIMEType]++
	return nil
}

func (cfg *configuration) writeAttachment(outFile opsys.OutFile, att chatdb.Attachment) error {
	attPath, mimeType := att.Filepath, att.MIMEType
	if cfg.Options.OutputPDF {
		if jpgPath, err := cfg.OS.HEIC2JPG(attPath); err != nil {
			cfg.counts.conversionsFailed++
			log.Printf("WARN: chat file %q - convert HEIC file %q to JPG: %s", outFile.Name(), attPath, err)
		} else if jpgPath != attPath {
			cfg.counts.conversions++
			attPath, mimeType = jpgPath, "image/jpeg"
		}
	}
	embedded, err := outFile.WriteAttachment(attPath)
	if err != nil {
		return errors.Wrapf(err, "include attachment %q", attPath)
	}
	if embedded {
		cfg.counts.attachmentsEmbedded[mimeType]++
	}
	cfg.counts.attachments[mimeType]++
	return nil
}
