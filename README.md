# bagoup [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov] [![Go Report Card][card-img]][card]

bagoup *(pronounced BAAGoop)* is an export utility for Mac OS Messages,
implemented in Go, inspired by
[Baskup](http://peterkaminski09.github.io/baskup/). It exports all of the
conversations saved in Messages to readable, searchable text files.

## "Installation"
In your GOPATH:

`git clone git@github.com:tagatac/bagoup.git`

## Usage
### chat.db Access
The Messages database is a protected file in Mac OS. See
[this article](https://appletoolbox.com/seeing-error-operation-not-permitted-in-macos-mojave/)
for more details. To to backup your messages, you have two options:
#### Option 1 (recommended): Copy chat.db
Copy the Messages database to an unprotected folder outside of the terminal.

1. Open Finder.
1. Navigate to **~/Library/Messages**.
1. Right-click on **chat.db**, and click **Copy "chat.db"** in the context menu.
1. Navigate to your clone of this repo.
1. Right-click in the `bagoup` directory, and click **Paste Item** in the
context menu.

#### Option 2: Give your terminal full disk access
https://osxdaily.com/2018/10/09/fix-operation-not-permitted-terminal-error-macos/

### Export your contacts (optional)
If you provide your contacts, bagoup will attempt to match the handles from the
Messages database with full names from your contacts list.

1. Export your contacts as a vCard file from e.g. the Contacts app or Google
Contacts
1. Copy the file to the `bagoup` directory as **contacts.vcf**.

### Build
`make`

### Run
```
Usage:
  bagoup [OPTIONS]

Application Options:
  -d, --db-path=        Path to the Messages chat database file (default: ~/Library/Messages/chat.db)
  -v, --mac-os-version= Version of Mac OS from which the Messages chat database file was copied
  -c, --contacts-path=  Path to the contacts vCard file
  -o, --export-path=    Path to which the Messages will be exported (default: backup)
  -h, --self-handle=    Prefix to use for for messages sent by you (default: Me)

Help Options:
  -h, --help            Show this help message
```
All conversations will be exported as text files to the specified export path.

## Author
Copyright (C) 2020 [David Tagatac](mailto:david@tagatac.net)

[ci-img]: https://travis-ci.org/tagatac/bagoup.svg?branch=master
[ci]: https://travis-ci.org/tagatac/bagoup
[cov-img]: https://codecov.io/gh/tagatac/bagoup/branch/master/graph/badge.svg
[cov]: https://codecov.io/gh/tagatac/bagoup
[card-img]: https://goreportcard.com/badge/github.com/tagatac/bagoup
[card]: https://goreportcard.com/report/github.com/tagatac/bagoup
