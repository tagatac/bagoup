package chatdb

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	_TypedStreamAttributeRE          = regexp.MustCompile(`(\{\n)? {4}"__kIM[[:alpha:]]+" = ([^\n]+);\n\}?`)
	_TypedStreamMultilineAttributeRE = regexp.MustCompile(`(\{\n)? {4}"__kIM[[:alpha:]]+" = {5}\{\n( {8}[[:alpha:]]+ = [\w-"]+;\n)+ {4}\};\n\}?`)
)

func (d *chatDB) decodeTypedStream(s string) (string, error) {
	cmd := d.execCommand("typedstream-decode")
	cmd.Stdin = bytes.NewReader([]byte(s))
	decodedBodyBytes, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "decode attributedBody - POSSIBLE FIX: Add typedstream-decode to your system path (installed with bagoup)")
	}
	decodedBody := string(decodedBodyBytes)
	decodedBody = _TypedStreamAttributeRE.ReplaceAllString(decodedBody, "")
	decodedBody = _TypedStreamMultilineAttributeRE.ReplaceAllString(decodedBody, "")
	return strings.TrimSuffix(decodedBody, "\n"), nil
}
