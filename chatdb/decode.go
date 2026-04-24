package chatdb

import (
	"fmt"
	"strings"

	ts "github.com/tagatac/typedstream-go"
)

func (d *chatDB) decodeTypedStream(s string) (string, error) {
	u, err := ts.NewUnarchiverFromData([]byte(s))
	if err != nil {
		return "", fmt.Errorf("create unarchiver: %w", err)
	}
	groups, err := u.DecodeAll()
	if err != nil {
		return "", fmt.Errorf("decode all: %w", err)
	}
	// The top-level group has one value: the NSMutableAttributedString object.
	if len(groups) == 0 || len(groups[0].Values) == 0 {
		return "", fmt.Errorf("empty stream")
	}
	obj, ok := groups[0].Values[0].(*ts.GenericArchivedObject)
	if !ok {
		return "", fmt.Errorf("unexpected top-level type %T", groups[0].Values[0])
	}
	// The first content group holds the NSMutableString / NSString.
	if len(obj.Contents) == 0 || len(obj.Contents[0].Values) == 0 {
		return "", fmt.Errorf("no string content")
	}
	var baseStr string
	switch v := obj.Contents[0].Values[0].(type) {
	case *ts.NSMutableString:
		baseStr = v.Value
	case *ts.NSString:
		baseStr = v.Value
	default:
		return "", fmt.Errorf("unexpected string type %T", obj.Contents[0].Values[0])
	}

	// Scan remaining content groups for an NSDictionary containing attributes.
	// Collect key=value pairs for keys that do NOT start with "__kIM", mirroring
	// the Darwin regex that strips __kIM* keys and keeps the rest (e.g. IMAudioTranscription).
	// Format in ObjC description style for cross-platform consistency.
	var extras []string
	for _, group := range obj.Contents[1:] {
		for _, val := range group.Values {
			dict, ok := val.(*ts.NSDictionary)
			if !ok {
				continue
			}
			for _, kv := range dict.Contents {
				k, ok := kv.Key.(*ts.NSString)
				if !ok || strings.HasPrefix(k.Value, "__kIM") {
					continue
				}
				if v, ok := kv.Value.(*ts.NSString); ok {
					extras = append(extras, fmt.Sprintf(`    %s = "%s"`, k.Value, v.Value))
				}
			}
		}
	}
	if len(extras) > 0 {
		return baseStr + "{\n" + strings.Join(extras, "\n") + "\n}", nil
	}
	return baseStr, nil
}
