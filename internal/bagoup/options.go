package bagoup

import "github.com/pkg/errors"

type // Options are the commandline options that can be passed to the bagoup
// command.
Options struct {
	DBPath          string   `short:"i" long:"db-path" description:"Path to the Messages chat database file" default:"~/Library/Messages/chat.db"`
	ExportPath      string   `short:"o" long:"export-path" description:"Path to which the Messages will be exported" default:"messages-export"`
	MacOSVersion    *string  `short:"m" long:"mac-os-version" description:"Version of Mac OS, e.g. '10.15', from which the Messages chat database file was copied (not needed if bagoup is running on the same Mac)"`
	ContactsPath    *string  `short:"c" long:"contacts-path" description:"Path to the contacts vCard file"`
	SelfHandle      string   `short:"s" long:"self-handle" description:"Prefix to use for for messages sent by you" default:"Me"`
	SeparateChats   bool     `long:"separate-chats" description:"Do not merge chats with the same contact (e.g. iMessage and SMS) into a single file"`
	OutputPDF       bool     `short:"p" long:"pdf" description:"Export text and images to PDF files (requires full disk access)"`
	IncludePPA      bool     `long:"include-ppa" description:"Include plugin payload attachments (e.g. link previews) in generated PDFs"`
	CopyAttachments bool     `short:"a" long:"copy-attachments" description:"Copy attachments to the same folder as the chat which included them (requires full disk access)"`
	PreservePaths   bool     `short:"r" long:"preserve-paths" description:"When copying attachments, preserve the full path instead of co-locating them with the chats which included them"`
	AttachmentsPath string   `short:"t" long:"attachments-path" description:"Root path to the attachments (useful for re-running bagoup on an export created with the --copy-attachments and --preserve-paths flags)" default:"/"`
	Entities        []string `short:"e" long:"entity" description:"An entity name to include in the export (matches the folder name in the export, e.g. \"John Smith\" or \"+15551234567\"). If given, other entities' chats will not be exported. If this flag is used multiple times, all entities specified will be exported."`
	PrintVersion    bool     `short:"v" long:"version" description:"Show the version of bagoup"`
}

func ValidateOptions(opts Options) error {
	if opts.IncludePPA && !opts.OutputPDF {
		return errors.New("the --include-ppa flag requires the --pdf flag")
	}
	if opts.PreservePaths && !opts.CopyAttachments {
		return errors.New("the --preserve-paths flag requires the --copy-attachments flag")
	}
	usingAttachments := opts.CopyAttachments || opts.OutputPDF
	if opts.AttachmentsPath != "/" && !usingAttachments {
		return errors.New("the --attachments-path flag requires a flag that uses those attachments: --copy-attachments or --pdf")
	}
	return nil
}
